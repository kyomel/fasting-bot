package usecase

import (
	"errors"
	"fmt"
	"log"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

func (u *fastingUsecase) CancelToday(phone string) (string, error) {
	now := time.Now().In(config.Location)
	return u.finishFastAt(phone, now, true)
}

func (u *fastingUsecase) BreakFastingAt(phone, dateInput, openTime string) (string, error) {
	openedAt, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+openTime, config.Location)
	if err != nil {
		return "❌ Format /buka salah. Gunakan: /buka DD-MM-YYYY HH:MM\nContoh: /buka 23-05-2026 18:30", nil
	}

	now := time.Now().In(config.Location).Truncate(time.Minute)
	if openedAt.After(now) {
		return "❌ Waktu buka tidak boleh di masa depan.", nil
	}

	return u.finishFastAt(phone, openedAt, false)
}

func (u *fastingUsecase) finishFastAt(phone string, openedAt time.Time, allowCancelBeforeStart bool) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	schedule, err := u.lookupActiveSchedule(user.ID)
	if err != nil {
		return "", err
	}
	if schedule == nil {
		return "ℹ️ Belum ada jadwal fasting aktif untuk dibuka.", nil
	}

	startTime, _ := parseScheduleTime(schedule.FastStart, openedAt)
	plannedEndTime, _ := parseScheduleTime(schedule.FastEnd, openedAt)
	if !plannedEndTime.After(startTime) {
		plannedEndTime = plannedEndTime.AddDate(0, 0, 1)
	}
	if openedAt.Before(startTime) {
		if !allowCancelBeforeStart {
			return fmt.Sprintf("❌ Waktu buka tidak boleh sebelum puasa mulai.\nMulai: %s\nBuka yang kamu input: %s", formatDisplayTime(startTime), formatDisplayTime(openedAt)), nil
		}
		return u.cancelBeforeStart(user.ID, startTime)
	}
	return u.breakFasting(user, schedule, startTime, plannedEndTime, openedAt)
}

func (u *fastingUsecase) lookupActiveSchedule(userID domain.ID) (*domain.FastingSchedule, error) {
	schedule, err := u.scheduleRepo.FindActiveByUserID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal memeriksa jadwal: %w", err)
	}
	return schedule, nil
}

func (u *fastingUsecase) cancelBeforeStart(userID domain.ID, startTime time.Time) (string, error) {
	if err := u.scheduleRepo.DeactivateByUserID(userID); err != nil {
		return "", fmt.Errorf("gagal membatalkan jadwal: %w", err)
	}
	if err := u.notificationRepo.LogNotification(userID, "cancelled"); err != nil {
		return "", fmt.Errorf("gagal mencatat pembatalan: %w", err)
	}
	return fmt.Sprintf("ℹ️ Jadwal fasting dibatalkan.\nMulai: %s\n\nKarena /buka dilakukan sebelum jam puasa mulai, durasi tidak dihitung ke /stats.", formatDisplayTime(startTime)), nil
}

func (u *fastingUsecase) breakFasting(user *domain.User, schedule *domain.FastingSchedule, startTime, plannedEndTime, openedAt time.Time) (string, error) {
	durationMinutes := int(openedAt.Sub(startTime).Minutes())
	if durationMinutes < 0 {
		durationMinutes = 0
	}
	streakQualified := !openedAt.Before(plannedEndTime)

	record := &domain.FastingRecord{
		UserID:          user.ID,
		ScheduleID:      schedule.ID,
		FastingTypeName: schedule.FastingTypeName,
		FastStart:       schedule.FastStart,
		PlannedFastEnd:  schedule.FastEnd,
		OpenedAt:        formatStoredTime(openedAt),
		DurationMinutes: durationMinutes,
		CompletedDate:   openedAt.Format("2006-01-02"),
		StreakQualified: streakQualified,
	}
	if err := u.scheduleRepo.CreateFastingRecord(record); err != nil {
		return "", fmt.Errorf("gagal menyimpan hasil buka puasa: %w", err)
	}
	if err := u.scheduleRepo.UpsertFastingStats(record); err != nil {
		return "", fmt.Errorf("gagal memperbarui stats puasa: %w", err)
	}
	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menutup jadwal: %w", err)
	}
	if err := u.notificationRepo.LogNotification(user.ID, "opened"); err != nil {
		return "", fmt.Errorf("gagal membatalkan: %w", err)
	}
	newBadges, err := u.evaluateAndAwardBadges(user.ID, record)
	if err != nil {
		log.Printf("[WARN] evaluateAndAwardBadges failed for user %s: %v", user.ID, err)
	}
	badgeUnlockMessage := formatBadgeUnlock(newBadges)

	durationHours := durationMinutes / 60
	extendedFastMessage := extendedFastCompletionMessage(durationHours, schedule.FastingTypeName)
	if streakQualified {
		message := fmt.Sprintf(
			"🎊 *Selesai — keren banget!*\n"+
				"Jenis: *%s*\n"+
				"⏱ Mulai: %s\n"+
				"🏁 Target: %s\n"+
				"🍽 Buka: %s\n"+
				"⌛ Total: *%s*\n\n"+
				"🔥 Streak bertambah! Adaptasi metabolik kamu makin solid setiap sesi.\n\n"+
				"%s"+
				"🥗 *Cara buka yang ramah tubuh:*\n%s\n\n"+
				"Cek progress: */stats* (pribadi) atau */leaderboard* (grup) 🏆",
			displayFastingTypeName(schedule.FastingTypeName),
			formatDisplayTime(startTime),
			formatDisplayTime(plannedEndTime),
			formatDisplayTime(openedAt),
			formatDurationWithDays(durationMinutes),
			extendedFastMessage,
			breakFastTipForDuration(durationHours),
		)
		if badgeUnlockMessage != "" {
			message += "\n\n" + badgeUnlockMessage
		}
		return message, nil
	}

	message := fmt.Sprintf(
		"✅ *Buka puasa tercatat!*\n"+
			"Jenis: *%s*\n"+
			"⏱ Mulai: %s\n"+
			"🏁 Target: %s\n"+
			"🍽 Buka: %s\n"+
			"⌛ Durasi: *%s*\n\n"+
			"Buka sebelum target — streak belum naik, tapi durasi tetap masuk stats. Semua jam puasa = waktu insulin rendah = manfaat tetap.\n\n"+
			"%s"+
			"🥗 *Cara buka yang ramah tubuh:*\n%s\n\n"+
			"Konsistensi > kesempurnaan. Set jadwal berikutnya: /puasa atau /puasa-dry 🌱",
		displayFastingTypeName(schedule.FastingTypeName),
		formatDisplayTime(startTime),
		formatDisplayTime(plannedEndTime),
		formatDisplayTime(openedAt),
		formatDurationWithDays(durationMinutes),
		extendedFastMessage,
		breakFastTipForDuration(durationHours),
	)
	if badgeUnlockMessage != "" {
		message += "\n\n" + badgeUnlockMessage
	}
	return message, nil
}

func extendedFastCompletionMessage(durationHours int, fastingTypeName string) string {
	if durationHours < 24 {
		return ""
	}
	if isDryFasting(fastingTypeName) {
		return "🧬 *Extended fast selesai!*\nKamu sudah melewati zona 24+ jam: insulin rendah, autophagy tinggi, dan mental endurance terlatih. Pulihkan tubuh bertahap dan jangan langsung makan besar.\n\n"
	}
	return "🧬 *Extended fast selesai!*\nKamu sudah masuk zona 24+ jam: autophagy tinggi, keton dominan, dan tubuh bekerja dalam mode repair. Refeed pelan-pelan; perhatikan elektrolit Na/K/Mg agar transisinya aman.\n\n"
}

// breakFastTipForDuration returns short, secular refeeding advice scaled to
// the actual fast length (insulin-spike avoidance for short fasts up to
// refeeding-syndrome warnings for prolonged fasts).
func breakFastTipForDuration(durationHours int) string {
	switch {
	case durationHours < 12:
		return "• Mulai dengan protein + lemak sebelum karbo — kurva insulin lebih landai.\n• Makan pelan, kasih waktu hormon kenyang (GLP-1, CCK) bekerja."
	case durationHours < 18:
		return "• Protein & lemak dulu (telur, alpukat, kacang), baru sayur, baru karbo.\n• Hindari gula/nasi sebagai gigitan pertama — bisa bikin reactive hypoglycemia.\n• 1-2 gelas air dulu sebelum makan."
	case durationHours < 24:
		return "• Mulai dengan kaldu hangat atau sup ringan — bangunkan sistem cerna pelan.\n• Lalu protein (telur/yogurt), baru karbo kompleks.\n• Tambahkan sejumput garam — natrium banyak terbuang saat puasa panjang."
	case durationHours < 48:
		return "• *Cairan dulu 30-60 menit:* kaldu tulang, air kelapa, sup sayur — bukan langsung makanan padat.\n• Lalu makanan lunak: telur rebus, pisang, alpukat.\n• Hindari: gorengan, gula, alkohol, sayur mentah.\n• Elektrolit penting: garam + kalium duluan."
	default:
		return "⚠️ *Refeed pelan-pelan — refeeding syndrome itu nyata di puasa >48 jam:*\n" +
			"• 4-6 jam pertama: cairan saja (kaldu + elektrolit Na/K/Mg).\n" +
			"• Lalu jus encer / bubur tipis / pisang lembut.\n" +
			"• Makanan utuh baru setelah 24 jam refeed bertahap.\n" +
			"• Hindari porsi besar — fosfat & magnesium bisa drop drastis kalau langsung makan banyak."
	}
}
