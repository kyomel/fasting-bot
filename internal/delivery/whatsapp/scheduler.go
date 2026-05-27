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
		msg := fmt.Sprintf(
			"⏰ *Puasa dimulai, %s!*\n\n"+
				"🏁 Target buka: *%s*\n\n"+
				"Tubuhmu mulai bekerja — insulin turun, sel masuk mode repair.\n"+
				"Jam pertama memang paling berat, tapi kamu sudah buktikan bisa lewati itu. 💪\n\n"+
				"💧 Minum air putih tetap boleh\n"+
				"🧘 Kalau berat, tarik napas dalam 4 detik — tahan 4 — buang 4\n\n"+
				"Let's go! 🔥",
			t.Name, formatScheduleForMessage(t.FastEnd),
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
		streakMsg := buildStreakMessage(t.Name, t.CurrentStreakDays)
		msg := fmt.Sprintf(
			"🏁 *Waktunya buka, %s!*\n\n"+
				"Puasa dari *%s* sampai *%s*\n"+
				"⌛ Total: *%s*\n\n"+
				"%s\n\n"+
				"Catat buka puasamu:\n"+
				"• */buka* → buka sekarang\n"+
				"• */buka %s HH:MM* → buka di waktu lain\n\n"+
				"_Tanpa /buka, durasi nggak masuk stats!_ ⚠️",
			t.Name,
			formatScheduleForMessage(t.FastStart),
			formatScheduleForMessage(t.FastEnd),
			duration,
			streakMsg,
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
		"Setelah 12 jam puasa, tubuh mulai beralih dari glukosa ke lemak sebagai sumber energi utama.",
		"Autophagy — proses daur ulang sel rusak — mulai aktif setelah 16-18 jam puasa.",
		"Intermittent fasting terbukti meningkatkan produksi BDNF, protein yang memperkuat daya ingat dan fokus.",
		"Puasa memberi waktu istirahat pada sistem pencernaan, sehingga energi bisa dialihkan untuk repair sel.",
		"Riset menunjukkan fasting teratur bisa meningkatkan sensitivitas insulin dan stabilitas energi sepanjang hari.",
	}
	tip := tips[time.Now().YearDay()%len(tips)]

	return fmt.Sprintf(
		"🌤️ *Sore Check-in*\n\n"+
			"Belum ada yang puasa hari ini!\n\n"+
			"Sore ini waktu bagus untuk mulai — setelah makan siang, tubuhmu siap masuk mode recovery.\n\n"+
			"💡 *Tahukah kamu?*\n%s\n\n"+
			"Siapa yang mau mulai? → /set-puasa atau /list-puasa 💪",
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

	return fmt.Sprintf(
		"🌤️ *Sore Check-in*\n\n"+
			"%s sedang berjuang sekarang! 🔥\n\n"+
			"%s\n\n"+
			"Semangat terus — setiap menit yang lewat, tubuhmu makin kuat. 💪\n\n"+
			"Yang belum mulai, masih bisa ikutan! → /set-puasa",
		countWord,
		strings.Join(lines, "\n"),
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
				"Streak putus bukan berarti gagal. Tubuhmu masih menyimpan semua progress sebelumnya.\n"+
				"Yang penting: bangkit dan mulai lagi.\n\n"+
				"Restart kapan saja → /set-puasa 💪",
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

func buildStreakMessage(name string, currentStreakDays int) string {
	switch {
	case currentStreakDays <= 0:
		return "🌱 Ini bisa jadi awal streak baru!"
	case currentStreakDays <= 2:
		return fmt.Sprintf("🌱 *Streak %s: %d hari*\nLangkah pertama sudah diambil — ini yang paling penting!", name, currentStreakDays)
	case currentStreakDays <= 6:
		return fmt.Sprintf("🌿 *Streak %s: %d hari!*\nKebiasaan mulai terbentuk — tubuhmu mulai beradaptasi, autophagy makin efisien!", name, currentStreakDays)
	case currentStreakDays <= 13:
		return fmt.Sprintf("🔥 *Streak %s: %d hari!*\nSatu minggu lebih! Metabolismemu sudah mulai berubah. Konsistensi level pro!", name, currentStreakDays)
	case currentStreakDays <= 29:
		return fmt.Sprintf("⚡ *Streak %s: %d hari!*\nDua minggu lebih nonstop — ini bukan lagi coba-coba, ini sudah jadi gaya hidup!", name, currentStreakDays)
	default:
		return fmt.Sprintf("👑 *Streak %s: %d hari!*\nSebulan lebih! Level legend. Konsistensi luar biasa — terus jaga momentum ini!", name, currentStreakDays)
	}
}
