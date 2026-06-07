package whatsapp

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/repository"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	scheduleRepo repository.ScheduleRepository
	notifRepo    repository.NotificationRepository
	notifier     *whatsapp.Notifier
	log          *slog.Logger
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
		log:          slog.Default().With("component", "scheduler"),
	}
}

func (s *Scheduler) Start() {
	// Cron splits:
	//   - start/end notifications: every minute (precise boundary)
	//   - proactive motivations (phase/hydration/near target): every 5 minutes
	//     to cut DB load without losing meaningful precision (windows are 1h+ wide)
	s.cron = cron.New(
		cron.WithLocation(config.Location),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)
	s.cron.AddFunc("* * * * *", s.checkStartEnd)
	s.cron.AddFunc("*/5 * * * *", s.checkProactiveMotivations)
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

// checkStartEnd runs at 1-minute precision: start/end are fasting boundaries
// that should fire within a minute of the actual scheduled time.
func (s *Scheduler) checkStartEnd() {
	now := time.Now().In(config.Location)
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")
	currentDateTime := now.Format("2006-01-02 15:04")

	s.notifyStart(currentTime, currentDate, currentDateTime)
	s.notifyEnd(currentTime, currentDate, currentDateTime, now)
}

// checkProactiveMotivations runs every 5 minutes: phase milestones, near-target
// nudges, and hydration reminders all use 1-hour+ windows so 5-min cadence is
// indistinguishable from 1-min cadence for the user, while cutting DB load ~5x.
func (s *Scheduler) checkProactiveMotivations() {
	now := time.Now().In(config.Location)
	currentDateTime := now.Format("2006-01-02 15:04")
	s.notifyProactiveMotivations(currentDateTime, now)
}

func (s *Scheduler) notifyStart(currentTime, currentDate, currentDateTime string) {
	targets, err := s.scheduleRepo.FindUsersToNotifyStart(currentTime, currentDate, currentDateTime)
	if err != nil {
		s.log.Warn("scheduler error (start)", "error", err)
		return
	}

	for _, t := range targets {
		durationHours := fastDurationHours(t.FastStart, t.FastEnd)
		preview := fastingMilestonePreview(durationHours)
		if isDryFastingName(t.FastingTypeName) {
			preview = dryFastingPreview(durationHours)
		}
		closing := "🧘 Lapar datang bergelombang. Minum yang cukup, napas pelan, lanjut satu jam lagi. 🔥"
		if isDryFastingName(t.FastingTypeName) {
			closing = "🧘 Lapar datang bergelombang. Napas pelan, pantau tubuh, jangan paksa kalau tidak aman. 🔥"
		}
		msg := fmt.Sprintf(
			"⏰ *Puasa dimulai, %s!*\n\n"+
				"🏁 Target: *%s* (±%d jam)\n"+
				"%s\n\n"+
				"%s\n\n"+
				"%s",
			t.Name,
			formatScheduleForMessage(t.FastEnd),
			durationHours,
			preview,
			startSafetyMessage(t.FastingTypeName, durationHours),
			closing,
		)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			s.log.Warn("failed to send start notification", "user", t.Name, "error", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "start"); err != nil {
			s.log.Warn("failed to log start notification", "user", t.Name, "error", err)
			continue
		}
		s.log.Info("📨 sent start notification", "user", t.Name)
	}
}

// --- Proactive motivational notifications ---

func (s *Scheduler) notifyProactiveMotivations(currentDateTime string, now time.Time) {
	s.notifyPhaseMilestones(currentDateTime)
	s.notifyNearTargetMotivations(currentDateTime)
	s.notifyHydrationReminders(currentDateTime, now)
}

func (s *Scheduler) notifyPhaseMilestones(currentDateTime string) {
	for _, trigger := range domain.ProactivePhaseNotifications {
		targets, err := s.scheduleRepo.FindUsersForElapsedNotification(trigger.NotificationType, trigger.TriggerAfterHours, currentDateTime)
		if err != nil {
			s.log.Warn("scheduler error (phase)", "type", trigger.NotificationType, "error", err)
			continue
		}

		for _, t := range targets {
			msg := buildPhaseMilestoneMessage(t, trigger, currentDateTime)
			s.sendAndLog(t, trigger.NotificationType, msg)
		}
	}
}

func (s *Scheduler) notifyNearTargetMotivations(currentDateTime string) {
	targets, err := s.scheduleRepo.FindUsersNearTargetNotification(domain.NotificationTypeNearTarget, currentDateTime)
	if err != nil {
		s.log.Warn("scheduler error (near target)", "error", err)
		return
	}

	for _, t := range targets {
		msg := buildNearTargetMotivationMessage(t, currentDateTime)
		s.sendAndLog(t, domain.NotificationTypeNearTarget, msg)
	}
}

func (s *Scheduler) notifyHydrationReminders(currentDateTime string, now time.Time) {
	for _, hour := range domain.HydrationReminderHours {
		notificationType := domain.HydrationNotificationType(hour)
		targets, err := s.scheduleRepo.FindUsersForElapsedNotification(notificationType, hour, currentDateTime)
		if err != nil {
			s.log.Warn("scheduler error (hydration)", "type", notificationType, "error", err)
			continue
		}

		for _, t := range targets {
			remaining, ok := remainingDuration(t.FastEnd, now)
			if isDryFastingName(t.FastingTypeName) || !ok || remaining <= 2*time.Hour {
				continue
			}

			msg := buildHydrationReminderMessage(t, hour, currentDateTime)
			s.sendAndLog(t, notificationType, msg)
		}
	}
}

func (s *Scheduler) sendAndLog(target repository.NotificationTarget, notificationType, msg string) {
	if err := s.notifier.Send(target.JID, msg); err != nil {
		s.log.Warn("failed to send notification", "type", notificationType, "user", target.Name, "error", err)
		return
	}
	if err := s.notifRepo.LogNotification(target.UserID, notificationType); err != nil {
		s.log.Warn("failed to log notification", "type", notificationType, "user", target.Name, "error", err)
		return
	}
	s.log.Info("📨 sent notification", "type", notificationType, "user", target.Name)
}

func buildPhaseMilestoneMessage(t repository.NotificationTarget, trigger domain.ProactivePhaseNotification, currentDateTime string) string {
	phase := domain.PhaseForElapsedHours(float64(trigger.TriggerAfterHours))
	body := schedulerMessage(domain.MotivationForPhase(trigger.PhaseKey), t.UserID)
	if nudge := proactiveSafetyNudge(t.FastingTypeName, trigger.TriggerAfterHours, t.UserID); nudge != "" {
		body += "\n\n" + nudge
	}

	return fmt.Sprintf(
		"%s *%s aktif, %s!*\n\n"+
			"Puasa sekitar *%d jam* — sisa *%s*.\n\n"+
			"%s\n\n"+
			"Progress kecil tetap progress. Dengarkan tubuh, lanjut bila terasa aman. 💪",
		phase.Emoji,
		phase.Name,
		t.Name,
		trigger.TriggerAfterHours,
		calculateDuration(currentDateTime, t.FastEnd),
		body,
	)
}

func buildNearTargetMotivationMessage(t repository.NotificationTarget, currentDateTime string) string {
	elapsed := calculateDuration(t.FastStart, currentDateTime)
	remaining := calculateDuration(currentDateTime, t.FastEnd)
	body := schedulerMessage(domain.MotivationNearTarget(), t.UserID)
	if nudge := proactiveSafetyNudge(t.FastingTypeName, fastDurationHours(t.FastStart, currentDateTime), t.UserID); nudge != "" {
		body += "\n\n" + nudge
	}

	return fmt.Sprintf(
		"🏁 *Tinggal sedikit lagi, %s!*\n\n"+
			"⌛ Berjalan: *%s* • Sisa: *%s*\n\n"+
			"%s\n\n"+
			"Finish dengan tenang. Kalau tubuh tidak nyaman, keselamatan dulu. 🔥",
		t.Name,
		elapsed,
		remaining,
		body,
	)
}

func buildHydrationReminderMessage(t repository.NotificationTarget, elapsedHours int, currentDateTime string) string {
	body := schedulerMessage(domain.HydrationNudges(), t.UserID)
	if elapsedHours >= 24 && isWaterOrProlongedFastingName(t.FastingTypeName) {
		body += "\n\n" + schedulerMessage(domain.ElectrolyteNudges(), t.UserID)
	}

	return fmt.Sprintf(
		"💧 *Reminder hidrasi, %s*\n\n"+
			"Sudah sekitar *%d jam* puasa — sisa *%s*.\n\n"+
			"%s\n\n"+
			"Kadang lemas = cairan/elektrolit kurang, bukan lapar. Cek tubuh dulu. 💪",
		t.Name,
		elapsedHours,
		calculateDuration(currentDateTime, t.FastEnd),
		body,
	)
}

func (s *Scheduler) notifyEnd(currentTime, currentDate, currentDateTime string, now time.Time) {
	targets, err := s.scheduleRepo.FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime)
	if err != nil {
		s.log.Warn("scheduler error (end)", "error", err)
		return
	}

	todayDate := now.Format("02-01-2006")
	for _, t := range targets {
		duration := calculateDuration(t.FastStart, t.FastEnd)
		durationHours := fastDurationHours(t.FastStart, t.FastEnd)
		streakMsg := buildStreakMessage(t.Name, t.CurrentStreakDays)
		msg := fmt.Sprintf(
			"🏁 *Waktunya buka, %s!*\n\n"+
				"%s → %s\n"+
				"⌛ Total: *%s*\n\n"+
				"%s\n\n"+
				"🍽 *Cara buka yang ramah tubuh:*\n%s\n\n"+
				"📝 Catat agar masuk stats: */buka*\n"+
				"Kalau buka di waktu lain: `/buka %s 18:30`",
			t.Name,
			formatScheduleForMessage(t.FastStart),
			formatScheduleForMessage(t.FastEnd),
			duration,
			streakMsg,
			refeedGuidance(durationHours),
			todayDate,
		)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			s.log.Warn("failed to send end notification", "user", t.Name, "error", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "end"); err != nil {
			s.log.Warn("failed to log end notification", "user", t.Name, "error", err)
			continue
		}
		s.log.Info("📨 sent end notification", "user", t.Name)
	}
}

// --- Group notifications ---

func (s *Scheduler) sendGroupAfternoonUpdate() {
	now := time.Now().In(config.Location)
	currentDateTime := now.Format("2006-01-02 15:04")

	activeFasters, err := s.scheduleRepo.FindUsersWithActiveFasting(currentDateTime)
	if err != nil {
		s.log.Warn("scheduler error (afternoon update)", "error", err)
		return
	}

	// Skip the group message entirely when nobody is fasting — the daily
	// "no fasters" tip became noise once groups grew past a few users and
	// started delivering the same tip every afternoon to the same people.
	if len(activeFasters) == 0 {
		s.log.Info("⏭️ skipped group afternoon update (no active fasters)")
		return
	}

	msg := s.buildActiveFastersMessage(activeFasters, currentDateTime)
	if err := s.notifier.SendToGroup(msg); err != nil {
		s.log.Warn("failed to send group afternoon update", "error", err)
		return
	}
	s.log.Info("📨 sent group afternoon update", "active_fasters", len(activeFasters))
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
			"Sore bisa jadi start ringan: lewati makan malam, tidur lebih awal, besok tubuh sudah masuk ritme insulin rendah.\n\n"+
			"🧠 *Tahukah kamu?*\n%s\n\n"+
			"Mau mulai?\n"+
			"• */list-puasa* — lihat 10 jenis puasa\n"+
			"• */set-puasa <nomor> <jam>* — mulai hari ini",
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
		"Insulin sedang turun, akses lemak sedang naik. Keep it steady. 🔥",
		"Autophagy itu proses beres-beres sel. Pelan, sunyi, tapi nyata. ✨",
		"Ketone mulai jadi bahan bakar alternatif. Banyak orang merasa fokus lebih stabil. 🧠",
		"Puasa bukan cuma angka timbangan; ini latihan ritme makan dan istirahat tubuh. 💎",
		"Lapar datang seperti gelombang. Tunggu sebentar, biasanya reda. 💪",
	}
	encouragement := encouragements[time.Now().YearDay()%len(encouragements)]

	return fmt.Sprintf(
		"🌤️ *Sore Check-in*\n\n"+
			"%s sedang puasa sekarang! 🔥\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"Yang mau ikut: */set-puasa <nomor> <jam>*",
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
		s.log.Warn("scheduler error (broken streaks)", "error", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf(
			"🔄 *Streak Reset*\n\n"+
				"*%s* — streak %d hari telah reset.\n\n"+
				"Angka reset, progress tubuh tidak hilang. Mulai lagi dengan ritme yang lebih ringan kalau perlu.\n\n"+
				"Restart: */set-puasa <nomor> <jam>* 💪",
			t.Name, t.CurrentStreakDays,
		)

		// Send to user DM, not the group. Streak reset is a personal moment
		// and announcing it to the whole group created social-judgment noise
		// that didn't match the health-focused, secular framing of the bot.
		if err := s.notifier.Send(t.JID, msg); err != nil {
			s.log.Warn("failed to send streak broken notification", "user", t.Name, "error", err)
			continue
		}

		if err := s.scheduleRepo.ResetStreakByUserID(t.UserID); err != nil {
			s.log.Warn("failed to reset streak", "user", t.Name, "user_id", t.UserID, "error", err)
			continue
		}

		s.log.Info("📨 sent streak broken notification", "user", t.Name)
	}
}

// --- Helpers ---

func (s *Scheduler) cleanupFastingHistory() {
	cutoff := time.Now().In(config.Location).AddDate(0, 0, -3).Format("2006-01-02 15:04:05")
	deleted, err := s.scheduleRepo.CleanupOldFastingRecords(cutoff)
	if err != nil {
		s.log.Warn("failed to cleanup fasting history", "error", err)
		return
	}
	if deleted > 0 {
		s.log.Info("🧹 cleaned up old fasting history records", "deleted", deleted)
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

func remainingDuration(endStr string, now time.Time) (time.Duration, bool) {
	end, err := time.ParseInLocation("2006-01-02 15:04", endStr, config.Location)
	if err != nil {
		return 0, false
	}
	remaining := end.Sub(now)
	if remaining < 0 {
		return 0, true
	}
	return remaining, true
}

func schedulerMessage(pool []string, userID int64) string {
	if len(pool) == 0 {
		return "💪 Kamu sedang membangun konsistensi. Lanjutkan pelan-pelan, satu jam demi satu jam."
	}
	index := int((userID + int64(time.Now().In(config.Location).YearDay())) % int64(len(pool)))
	return pool[index]
}

func isDryFastingName(name string) bool {
	return strings.Contains(strings.ToLower(name), "dry fasting")
}

func isWaterOrProlongedFastingName(name string) bool {
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "water fasting") || strings.Contains(lowerName, "prolonged")
}

func proactiveSafetyNudge(fastingTypeName string, elapsedHours int, userID int64) string {
	if isDryFastingName(fastingTypeName) {
		return "⚠️ *Dry fasting:* jangan paksa tubuh. Kalau pusing berat, lemas ekstrem, bingung, atau terasa tidak aman, batalkan dengan bijak."
	}
	if elapsedHours >= 24 && isWaterOrProlongedFastingName(fastingTypeName) {
		return schedulerMessage(domain.ElectrolyteNudges(), userID)
	}
	return schedulerMessage(domain.HydrationNudges(), userID)
}

func startSafetyMessage(fastingTypeName string, durationHours int) string {
	if isDryFastingName(fastingTypeName) {
		return "⚠️ *Dry fasting:* pantau sinyal tubuh lebih ketat. Kalau pusing berat, lemas ekstrem, atau terasa tidak aman, batalkan dengan bijak."
	}
	if durationHours >= 24 && isWaterOrProlongedFastingName(fastingTypeName) {
		return "🧂 *Elektrolit wajib:* untuk water/prolonged fasting 24+ jam, perhatikan natrium, kalium, dan magnesium — jangan cuma air putih."
	}
	return "💧 *Tetap hidrasi:* air putih, kopi/teh tanpa gula tetap aman. Tambahkan sejumput garam kalau >16 jam — saat insulin turun, ginjal buang lebih banyak natrium."
}

func dryFastingPreview(durationHours int) string {
	if durationHours >= 24 {
		return "📈 *Yang akan terjadi di tubuhmu:*\n" +
			"• Insulin turun dan tubuh memakai cadangan energi lebih dalam\n" +
			"• Autophagy dan sinyal repair naik seiring durasi\n" +
			"⚠️ Dry fasting panjang itu serius — prioritaskan keselamatan dan dengarkan sinyal tubuh."
	}
	return "📈 *Yang akan terjadi di tubuhmu:*\n" +
		"• Insulin turun, glikogen mulai dipakai, dan tubuh masuk mode akses energi cadangan\n" +
		"• Rasa lapar datang bergelombang — biasanya reda jika kamu beri waktu."
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
