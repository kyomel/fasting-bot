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

func (u *fastingUsecase) GetStatus(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	name := user.Name
	if name == "" {
		name = user.Phone
	}

	schedule, err := u.scheduleRepo.FindActiveByUserID(user.ID)
	if err != nil {
		return fmt.Sprintf("📋 *Status Akun*\nID: %s\nNama: %s\nNomor: %s\n\nBelum ada jadwal puasa aktif.\n\n"+
			"Atur sekarang:\n"+
			"• */panduan* — baca panduan ringkas\n"+
			"• */puasa 16* — mulai 16 jam dari sekarang\n"+
			"• */puasa 16 23-05-2026 05:00* — jadwalkan ke tanggal tertentu", user.ID, name, user.Phone), nil
	}
	fastingTypeName := schedule.FastingTypeName
	if fastingTypeName == "" {
		fastingTypeName = "Belum diketahui"
	}

	now := time.Now().In(config.Location)
	startTime, startHasDate := parseScheduleTime(schedule.FastStart, now)
	endTime, endHasDate := parseScheduleTime(schedule.FastEnd, now)
	if !startHasDate && !endHasDate && !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	var status string
	if now.Before(startTime) {
		status = fmt.Sprintf("⏳ Fasting dimulai dalam %s", formatDuration(startTime.Sub(now)))
	} else if now.Before(endTime) {
		elapsed := now.Sub(startTime)
		phase := domain.PhaseForElapsedHours(elapsed.Hours())
		status = fmt.Sprintf(
			"🍽️ Sedang fasting!\nSudah berjalan: %s\nSisa: %s\nFase: %s *%s*",
			formatDuration(elapsed),
			formatDuration(endTime.Sub(now)),
			phase.Emoji,
			phase.Name,
		)
		if next, ok := phase.NextPhase(); ok {
			status += fmt.Sprintf("\nBerikutnya: %s %s (~jam ke-%.0f)", next.Emoji, next.Name, next.MinHours)
		}
	} else {
		elapsed := endTime.Sub(startTime)
		if now.After(endTime) {
			elapsed = now.Sub(startTime)
		}
		phase := domain.PhaseForElapsedHours(elapsed.Hours())
		status = fmt.Sprintf("✅ Target selesai!\nFase terakhir: %s *%s*\nJalankan */buka* untuk catat hasil.", phase.Emoji, phase.Name)
	}

	return fmt.Sprintf("📋 *Status Fasting*\nID: %s\nNama: %s\nNomor: %s\nJenis Puasa: %s\nMulai: %s\nSelesai: %s\n\n%s", user.ID, name, user.Phone, fastingTypeName, formatScheduleDisplay(schedule.FastStart), formatScheduleDisplay(schedule.FastEnd), status), nil
}

func (u *fastingUsecase) GetHistory(phone string, limit int) (string, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	records, err := u.scheduleRepo.FindRecentFastingRecords(user.ID, limit)
	if err != nil {
		return "", fmt.Errorf("gagal mengambil riwayat puasa: %w", err)
	}
	if len(records) == 0 {
		return "📜 *Riwayat Puasa*\nBelum ada sesi tercatat.\n\nSelesaikan satu sesi dengan */buka* dulu.", nil
	}

	name := displayUserName(user)
	result := fmt.Sprintf("📜 *Riwayat Puasa %s*\n%d sesi terakhir:\n\n", name, len(records))
	for i, record := range records {
		phase := domain.PhaseForElapsedHours(float64(record.DurationMinutes) / 60)
		result += fmt.Sprintf(
			"%d. *%s*\n   ⏱ %s → %s\n   ⌛ %s · %s %s\n",
			i+1,
			displayFastingTypeName(record.FastingTypeName),
			formatScheduleDisplay(record.FastStart),
			formatScheduleDisplay(record.OpenedAt),
			formatDurationWithDays(record.DurationMinutes),
			phase.Emoji,
			phase.Name,
		)
	}
	result += "\n_Ringkasan permanen di */stats*. Riwayat mentah dibersihkan tiap 3 hari._"
	return result, nil
}

func (u *fastingUsecase) GetStats(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}

	stats, err := u.scheduleRepo.FindFastingStatsByUserID(user.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
		}
		return "", fmt.Errorf("gagal mengambil stats: %w", err)
	}
	if stats.TotalSessions == 0 {
		return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
	}

	if _, err := u.evaluateAndAwardBadges(user.ID, nil); err != nil {
		log.Printf("[WARN] lazy badge backfill failed for user %s: %v", user.ID, err)
	}

	message := fmt.Sprintf("📊 *Stats Puasa %s*\nTotal sesi: %d\nStreak puasa saat ini: %d hari\nStreak puasa terpanjang: %d hari\nTotal waktu puasa: %s\n\nTerakhir buka: %s\nDurasi terakhir: %s", stats.Name, stats.TotalSessions, stats.CurrentStreakDays, stats.LongestStreakDays, formatDurationWithDays(stats.TotalMinutes), formatScheduleDisplay(stats.LastOpenedAt), formatDurationWithDays(stats.LastDurationMinutes))
	if shelf := u.badgeShelf(user.ID); shelf != "" {
		message += "\n\n🏅 *Badge:* " + shelf + "\nCek koleksi lengkap: /badge"
	}
	return message, nil
}

func (u *fastingUsecase) GetLeaderboard() (string, error) {
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}

	entries, err := u.scheduleRepo.FindFastingLeaderboard()
	if err != nil {
		return "", fmt.Errorf("gagal mengambil leaderboard: %w", err)
	}
	if len(entries) == 0 {
		return "🏆 *Leaderboard Puasa*\nBelum ada data puasa.\n\nLeaderboard akan terisi setelah user menjalankan /buka setelah puasa dimulai.", nil
	}
	if err := u.awardBadges(entries[0].UserID, []domain.BadgeKey{domain.BadgeGroupChampion}); err != nil {
		log.Printf("[WARN] award group champion badge failed for user %s: %v", entries[0].UserID, err)
	}

	limit := len(entries)
	if limit > 10 {
		limit = 10
	}

	result := "🏆 *Leaderboard Puasa*\nPatokan ranking: total waktu puasa\n\n"
	for i := 0; i < limit; i++ {
		entry := entries[i]
		result += fmt.Sprintf("%d. %s\n   Streak puasa: %d hari\n   Total: %s\n", i+1, entry.Name, entry.CurrentStreakDays, formatDurationWithDays(entry.TotalMinutes))
		if i < limit-1 {
			result += "\n"
		}
	}

	return result, nil
}

func (u *fastingUsecase) refreshStaleCurrentStreaks() error {
	now := time.Now().In(config.Location)
	return u.scheduleRepo.ResetStaleCurrentStreaks(now.Format("2006-01-02"), formatStoredTime(now))
}
