package whatsapp

import (
	"fmt"
	"strings"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/repository"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	scheduleRepo repository.ScheduleRepository
	notifRepo    repository.NotificationRepository
	notifier     *whatsapp.Notifier
}

func NewScheduler(
	scheduleRepo repository.ScheduleRepository,
	notifRepo repository.NotificationRepository,
	notifier *whatsapp.Notifier,
) *Scheduler {
	return &Scheduler{
		scheduleRepo: scheduleRepo,
		notifRepo:    notifRepo,
		notifier:     notifier,
	}
}

func (s *Scheduler) Start() {
	s.cron = cron.New(
		cron.WithLocation(config.Location),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)
	s.cron.AddFunc("* * * * *", s.checkAndNotify)
	s.cron.AddFunc("0 3 */3 * *", s.cleanupFastingHistory)
	s.cron.AddFunc("0 15 * * *", s.sendGroupAfternoonUpdate)
	s.cron.AddFunc("0 */4 * * *", s.checkBrokenStreaks)
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

// --- Personal notifications: start & end ---

func (s *Scheduler) checkAndNotify() {
	now := time.Now().In(config.Location)
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")
	currentDateTime := now.Format("2006-01-02 15:04")

	s.notifyStart(currentTime, currentDate, currentDateTime)
	s.notifyEnd(currentTime, currentDate, currentDateTime, now)
}

func (s *Scheduler) notifyStart(currentTime, currentDate, currentDateTime string) {
	targets, err := s.scheduleRepo.FindUsersToNotifyStart(currentTime, currentDate, currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (start): %v\n", err)
		return
	}

	for _, t := range targets {
		durationHours := fastDurationHours(t.FastStart, t.FastEnd)
		msg := fmt.Sprintf(
			"⏰ *Puasa dimulai, %s!*\n\n"+
				"🏁 Target buka: *%s* (±%d jam)\n\n"+
				"%s\n\n"+
				"💧 *Tetap hidrasi:* air putih, kopi/teh tanpa gula tetap aman. Tambahkan sejumput garam kalau >16 jam — saat insulin turun, ginjal buang lebih banyak natrium.\n"+
				"🧘 Lapar itu hormon (ghrelin), bukan keadaan darurat. Datang seperti gelombang, hilang dalam ~20 menit. Tarik napas 4-4-4 dan lewati.\n\n"+
				"_Kalau rencana berubah & mau ganti jadwal, pakai /set-puasa atau /jadwalkan._\n\n"+
				"Let's go! 🔥",
			t.Name,
			formatScheduleForMessage(t.FastEnd),
			durationHours,
			fastingMilestonePreview(durationHours),
		)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send start notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "start"); err != nil {
			fmt.Printf("❌ Failed to log start notification: %v\n", err)
			continue
		}
		fmt.Println("📨 Sent start notification")
	}
}

func (s *Scheduler) notifyEnd(currentTime, currentDate, currentDateTime string, now time.Time) {
	targets, err := s.scheduleRepo.FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (end): %v\n", err)
		return
	}

	todayDate := now.Format("02-01-2006")
	for _, t := range targets {
		duration := calculateDuration(t.FastStart, t.FastEnd)
		durationHours := fastDurationHours(t.FastStart, t.FastEnd)
		streakMsg := buildStreakMessage(t.Name, t.CurrentStreakDays)
		msg := fmt.Sprintf(
			"🏁 *Waktunya buka, %s!*\n\n"+
				"Puasa dari *%s* sampai *%s*\n"+
				"⌛ Total: *%s*\n\n"+
				"%s\n\n"+
				"🍽 *Cara buka yang ramah tubuh:*\n%s\n\n"+
				"📝 *Catat buka puasamu:*\n"+
				"• */buka* → buka sekarang\n"+
				"• */buka %s HH:MM* → buka di waktu lain (DD-MM-YYYY HH:MM)\n"+
				"  contoh: `/buka %s 18:30`\n\n"+
				"_Tanpa /buka, durasi nggak masuk stats & streak!_ ⚠️",
			t.Name,
			formatScheduleForMessage(t.FastStart),
			formatScheduleForMessage(t.FastEnd),
			duration,
			streakMsg,
			refeedGuidance(durationHours),
			todayDate,
			todayDate,
		)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send end notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "end"); err != nil {
			fmt.Printf("❌ Failed to log end notification: %v\n", err)
			continue
		}
		fmt.Println("📨 Sent end notification")
	}
}

// --- Group notifications ---

func (s *Scheduler) sendGroupAfternoonUpdate() {
	now := time.Now().In(config.Location)
	currentDateTime := now.Format("2006-01-02 15:04")

	activeFasters, err := s.scheduleRepo.FindUsersWithActiveFasting(currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (afternoon update): %v\n", err)
		return
	}

	var msg string
	if len(activeFasters) == 0 {
		msg = s.buildNoFastersMessage()
	} else {
		msg = s.buildActiveFastersMessage(activeFasters, currentDateTime)
	}

	if err := s.notifier.SendToGroup(msg); err != nil {
		fmt.Printf("❌ Failed to send group afternoon update: %v\n", err)
		return
	}
	fmt.Println("📨 Sent group afternoon update")
}

func (s *Scheduler) buildNoFastersMessage() string {
	tips := []string{
		"Sekitar jam ke-12 puasa, glikogen hati mulai habis dan tubuh menyalakan *metabolic switch* — beralih dari glukosa ke lemak sebagai bahan bakar utama (riset Mark Mattson, NIH).",
		"Autophagy — proses daur ulang sel rusak yang bikin Yoshinori Ohsumi dapat Nobel 2016 — mulai naik signifikan setelah ~16-18 jam puasa.",
		"Fasting memicu produksi BDNF (Brain-Derived Neurotrophic Factor), protein yang menjaga neuron tetap muda dan mempertajam fokus.",
		"Saat puasa, mTOR (sinyal pertumbuhan sel) turun & AMPK (sinyal repair) naik. Sel berhenti tumbuh dan mulai membereskan diri.",
		"Studi Satchin Panda (Salk Institute) menunjukkan time-restricted eating 12 jam saja sudah cukup memperbaiki sensitivitas insulin & jam biologis tubuh.",
		"Pada jam ke-24 puasa, hormon pertumbuhan (HGH) bisa naik sampai 20x lipat — itu mekanisme alami tubuh menjaga massa otot saat lemak dibakar.",
		"Ketone (BHB) yang diproduksi hati saat puasa bukan cuma bahan bakar — ia juga sinyal anti-inflamasi yang menekan inflammasome NLRP3.",
		"Riset Valter Longo (USC): puasa berkala menurunkan IGF-1, faktor pertumbuhan yang kalau berlebihan dikaitkan dengan penuaan dini & risiko penyakit kronis.",
		"Setelah 48 jam puasa, tubuh mulai mengirim sinyal regenerasi stem cell — sel imun lama dibersihkan, yang baru akan dibentuk saat refeed (Longo et al., Cell 2014).",
		"Insulin tinggi terus-menerus = sel \"tuli\" (insulin resistance). Jeda puasa = cara paling murah & alami buat \"reset\" pendengaran sel terhadap insulin.",
	}
	tip := tips[time.Now().YearDay()%len(tips)]

	return fmt.Sprintf(
		"🌤️ *Sore Check-in*\n\n"+
			"Belum ada yang puasa hari ini.\n\n"+
			"Sore = waktu bagus untuk mulai. Lewati makan malam, tidur dengan insulin rendah, dan besok pagi kamu sudah masuk mode fat-burning.\n\n"+
			"🧠 *Tahukah kamu?*\n%s\n\n"+
			"Siapa duluan? 💪\n"+
			"• */list-puasa* — lihat 10 jenis puasa\n"+
			"• */set-puasa <nomor> <jam>* — mulai hari ini\n"+
			"• */jadwalkan <nomor> <tanggal> <jam>* — kunci jadwal besok/lusa",
		tip,
	)
}

func (s *Scheduler) buildActiveFastersMessage(fasters []repository.NotificationTarget, currentDateTime string) string {
	var lines []string
	for _, f := range fasters {
		elapsed := calculateDuration(f.FastStart, currentDateTime)
		remaining := calculateDuration(currentDateTime, f.FastEnd)
		lines = append(lines, fmt.Sprintf("• *%s* — sudah %s, sisa %s", f.Name, elapsed, remaining))
	}

	countWord := fmt.Sprintf("%d orang", len(fasters))
	if len(fasters) == 1 {
		countWord = "1 orang"
	}

	encouragements := []string{
		"Setiap menit yang lewat, insulin makin rendah & lipolysis (pembakaran lemak) makin tinggi. Tubuh kalian lagi bekerja di mode yang tidak bisa dibeli di apotek mana pun. 🔥",
		"Sel kalian lagi sibuk autophagy — bersih-bersih protein rusak. Bayangin sel kalian lagi maraton beresin gudang. Hasilnya baru kelihatan minggu depan, tapi prosesnya jalan sekarang. ✨",
		"Otak kalian lagi switch ke ketone — bahan bakar yang lebih bersih daripada glukosa. Banyak yang bilang fokus malah naik di jam-jam ini. 🧠",
		"mTOR off, AMPK on. Mode growth dimatikan, mode repair dihidupkan. Ini bukan cuma soal angka di timbangan — ini soal kualitas sel kalian 5 tahun ke depan. 💎",
		"Lapar datang & pergi seperti gelombang — hormon ghrelin puncaknya cuma ~20 menit. Lewati gelombangnya, dia hilang sendiri. Kalian lebih kuat dari rasa lapar sesaat. 💪",
	}
	encouragement := encouragements[time.Now().YearDay()%len(encouragements)]

	return fmt.Sprintf(
		"🌤️ *Sore Check-in*\n\n"+
			"%s sedang puasa sekarang! 🔥\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"Yang belum ikut, pintu masih terbuka:\n"+
			"• */set-puasa <nomor> <jam>* — mulai sekarang\n"+
			"• */jadwalkan <nomor> <tanggal> <jam>* — set untuk besok",
		countWord,
		strings.Join(lines, "\n"),
		encouragement,
	)
}

func (s *Scheduler) checkBrokenStreaks() {
	now := time.Now().In(config.Location)
	currentDateTime := now.Format("2006-01-02 15:04")

	targets, err := s.scheduleRepo.FindUsersWithExpiredStreaks(currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (broken streaks): %v\n", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf(
			"🔄 *Streak Reset*\n\n"+
				"*%s* — streak %d hari telah reset.\n\n"+
				"Streak putus ≠ progress hilang. Adaptasi metabolik, sensitivitas insulin, dan fleksibilitas bahan bakar yang sudah kamu bangun *masih ada* di sel-selmu. Yang reset cuma angkanya, bukan biologinya.\n\n"+
				"Riset menunjukkan orang yang restart cepat setelah putus = orang yang punya streak panjang berikutnya. Yang membedakan bukan yang tidak pernah jatuh, tapi yang tidak lama di tanah.\n\n"+
				"Mulai lagi kapan pun:\n"+
				"• */set-puasa <nomor> <jam>* — mulai hari ini\n"+
				"• */jadwalkan <nomor> <tanggal> <jam>* — kunci jadwal besok\n"+
				"• */list-puasa* — pilih metode yang lebih ringan dulu (mis. IF 14:10) 💪",
			t.Name, t.CurrentStreakDays,
		)

		if err := s.notifier.SendToGroup(msg); err != nil {
			fmt.Printf("❌ Failed to send streak broken notification: %v\n", err)
			continue
		}

		if err := s.scheduleRepo.ResetStreakByUserID(t.UserID); err != nil {
			fmt.Printf("❌ Failed to reset streak for user %d: %v\n", t.UserID, err)
			continue
		}

		fmt.Printf("📨 Sent streak broken notification for %s\n", t.Name)
	}
}

// --- Helpers ---

func (s *Scheduler) cleanupFastingHistory() {
	cutoff := time.Now().In(config.Location).AddDate(0, 0, -3).Format("2006-01-02 15:04:05")
	deleted, err := s.scheduleRepo.CleanupOldFastingRecords(cutoff)
	if err != nil {
		fmt.Printf("❌ Failed to cleanup fasting history: %v\n", err)
		return
	}
	if deleted > 0 {
		fmt.Printf("🧹 Cleaned up %d old fasting history records\n", deleted)
	}
}

func formatScheduleForMessage(value string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04", value, config.Location)
	if err != nil {
		return value
	}
	return t.Format("02-01-2006 15:04")
}

func calculateDuration(startStr, endStr string) string {
	start, errS := time.ParseInLocation("2006-01-02 15:04", startStr, config.Location)
	end, errE := time.ParseInLocation("2006-01-02 15:04", endStr, config.Location)
	if errS != nil || errE != nil {
		return "-"
	}
	totalMinutes := int(end.Sub(start).Minutes())
	if totalMinutes < 0 {
		totalMinutes = 0
	}
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60
	totalHours := totalMinutes / 60
	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit (total: %d jam %d menit)", days, hours, minutes, totalHours, minutes)
	}
	return fmt.Sprintf("%d jam %d menit", hours, minutes)
}

// fastDurationHours returns the planned fast length in whole hours.
// Returns 0 if either timestamp can't be parsed.
func fastDurationHours(startStr, endStr string) int {
	start, errS := time.ParseInLocation("2006-01-02 15:04", startStr, config.Location)
	end, errE := time.ParseInLocation("2006-01-02 15:04", endStr, config.Location)
	if errS != nil || errE != nil {
		return 0
	}
	h := int(end.Sub(start).Hours())
	if h < 0 {
		return 0
	}
	return h
}

// fastingMilestonePreview returns a short, secular preview of physiological
// milestones the user will hit during a fast of the given length.
func fastingMilestonePreview(durationHours int) string {
	switch {
	case durationHours <= 0:
		return "Tubuhmu mulai bekerja — insulin turun, sel masuk mode repair."
	case durationHours <= 12:
		return "📈 *Yang akan terjadi di tubuhmu:*\n" +
			"• Jam 4-8: insulin turun, lipolysis (pembakaran lemak) mulai aktif\n" +
			"• Jam 10-12: glikogen hati menipis — metabolic switch mendekat\n" +
			"_Ini fondasi 12:12 — Satchin Panda (Salk) buktikan ini cukup untuk perbaiki sensitivitas insulin._"
	case durationHours <= 18:
		return "📈 *Yang akan terjadi di tubuhmu:*\n" +
			"• Jam 8-12: glikogen habis, *metabolic switch* aktif\n" +
			"• Jam 14-16: ketone (BHB) mulai terdeteksi, otak dapat bahan bakar bersih\n" +
			"• Jam 16-18: *autophagy* (Nobel 2016) mulai ramp-up — sel daur ulang sampah internal\n" +
			"_Ini sweet spot IF: ketosis ringan + masih bisa makan dalam jadwal normal._"
	case durationHours <= 24:
		return "📈 *Milestone yang akan kamu lewati:*\n" +
			"• Jam 12-16: metabolic switch + ketogenesis aktif\n" +
			"• Jam 16-20: autophagy & BDNF naik (fokus & mood biasanya membaik)\n" +
			"• Jam 20-24: HGH bisa naik berkali-kali lipat — proteksi alami massa otot\n" +
			"_Zona OMAD — single meal sehari, tubuh berjalan dominan dengan ketone._"
	case durationHours <= 48:
		return "📈 *Zona Water Fasting — milestone besar:*\n" +
			"• Jam 24: glikogen 0%, fat-adaptation penuh, autophagy puncak\n" +
			"• Jam 36: IGF-1 turun, mode \"damage control\" (Valter Longo)\n" +
			"• Jam 48: sinyal regenerasi stem cell — sel imun lama dibersihkan\n" +
			"⚠️ *Wajib elektrolit:* garam, kalium, magnesium. Jangan cuma air putih."
	default:
		return "📈 *Zona Prolonged Fasting — territory serius:*\n" +
			"• Jam 24-48: autophagy puncak, regenerasi stem cell mulai\n" +
			"• Jam 48-72: imun reset (Longo, Cell Stem Cell 2014/2019)\n" +
			"• Jam 72+: clearance protein misfolded, mitokondria diperbarui\n" +
			"⚠️ *Wajib:* hidrasi + elektrolit (Na, K, Mg) tiap hari. Pantau pusing/lemas berat.\n" +
			"⚠️ Refeed *pelan-pelan* nanti — risiko *refeeding syndrome* nyata di puasa >72 jam."
	}
}

// refeedGuidance returns short, science-based advice for breaking a fast of
// the given length — covering insulin response (short fasts) up to refeeding
// syndrome warnings (very long fasts).
func refeedGuidance(durationHours int) string {
	switch {
	case durationHours <= 0:
		return "• Mulai dengan protein + lemak sebelum karbo — bikin kurva insulin landai."
	case durationHours <= 18:
		return "• Protein & lemak dulu (telur, alpukat, kacang) — baru sayur, baru karbo\n" +
			"• Hindari nasi/roti/gula sebagai gigitan pertama — bisa bikin reactive hypoglycemia\n" +
			"• Makan pelan — hormon kenyang (GLP-1, CCK) butuh ~15 menit buat ngirim sinyal"
	case durationHours <= 24:
		return "• Mulai ringan: kaldu/sup hangat → telur/yogurt → baru makanan utuh\n" +
			"• Tambahkan sejumput garam ke kaldu (gantiin natrium yang terbuang)\n" +
			"• Tahan diri dari porsi besar — lambung kamu lagi tenang, kasih dia waktu"
	case durationHours <= 48:
		return "• *Cairan dulu (30-60 menit):* kaldu tulang, air kelapa, sup sayur — bukan langsung makanan padat\n" +
			"• Lalu makanan lunak: telur rebus, pisang, alpukat\n" +
			"• Hindari: makanan goreng, gula, alkohol, sayur mentah (sulit dicerna usus yang baru \"bangun\")\n" +
			"• Elektrolit penting: garam + kalium duluan, magnesium menyusul"
	default:
		return "⚠️ *Refeeding zone — pelan & bertahap:*\n" +
			"• 4-6 jam pertama: cairan saja (kaldu + elektrolit)\n" +
			"• Lalu jus encer / bubur tipis / pisang lembut\n" +
			"• Makanan utuh baru setelah 24 jam refeed bertahap\n" +
			"• *Refeeding syndrome itu nyata* — fosfat & magnesium bisa drop drastis kalau langsung makan banyak\n" +
			"• Aturan emas: makin panjang puasanya, makin lambat bukanya"
	}
}

func buildStreakMessage(name string, currentStreakDays int) string {
	switch {
	case currentStreakDays <= 0:
		return "🌱 Ini bisa jadi hari pertama streak baru — semua streak panjang dimulai dari hari ke-1."
	case currentStreakDays <= 2:
		return fmt.Sprintf("🌱 *Streak %s: %d hari*\nLangkah pertama paling berat secara neurologis — kamu sudah lewati itu. Otak mulai menulis ulang pola.", name, currentStreakDays)
	case currentStreakDays <= 6:
		return fmt.Sprintf("🌿 *Streak %s: %d hari!*\nMitokondria mulai naik level (mitochondrial biogenesis). Energi makin stabil, mood swing makin tipis. Adaptasi metabolik sedang dibentuk.", name, currentStreakDays)
	case currentStreakDays <= 13:
		return fmt.Sprintf("🔥 *Streak %s: %d hari!*\nSeminggu lebih! Tubuhmu sudah masuk fase *fat-adapted* — switch ke ketone makin cepat, autophagy makin efisien. Ini level yang banyak orang menyerah sebelum mencapainya.", name, currentStreakDays)
	case currentStreakDays <= 29:
		return fmt.Sprintf("⚡ *Streak %s: %d hari!*\nDua minggu nonstop. Sensitivitas insulin & metabolic flexibility kamu sekarang jauh lebih baik dari rata-rata. Bukan diet — sudah jadi sistem operasi tubuhmu.", name, currentStreakDays)
	case currentStreakDays <= 59:
		return fmt.Sprintf("👑 *Streak %s: %d hari!*\nSebulan lebih. IGF-1 mu stabil di zona longevity, ritme sirkadian mu makin rapi. Ini bukan lagi soal disiplin — ini soal identitas.", name, currentStreakDays)
	default:
		return fmt.Sprintf("🏆 *Streak %s: %d hari!*\nKamu sudah jadi case study hidup soal apa yang terjadi kalau manusia konsisten dengan biologinya sendiri. Hormat. 🫡", name, currentStreakDays)
	}
}
