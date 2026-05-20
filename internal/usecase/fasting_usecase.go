package usecase

import (
	"fmt"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type FastingUsecase interface {
	RegisterUser(phone, jid string) (string, error)
	SetSchedule(phone, start, end string) (string, error)
	GetStatus(phone string) (string, error)
	CancelToday(phone string) (string, error)
}

type fastingUsecase struct {
	userRepo         repository.UserRepository
	scheduleRepo     repository.ScheduleRepository
	notificationRepo repository.NotificationRepository
}

func NewFastingUsecase(
	userRepo repository.UserRepository,
	scheduleRepo repository.ScheduleRepository,
	notificationRepo repository.NotificationRepository,
) FastingUsecase {
	return &fastingUsecase{
		userRepo:         userRepo,
		scheduleRepo:     scheduleRepo,
		notificationRepo: notificationRepo,
	}
}

func (u *fastingUsecase) RegisterUser(phone, jid string) (string, error) {
	_, err := u.userRepo.FindByPhone(phone)
	if err == nil {
		return "✅ Kamu sudah terdaftar! Kirim /jadwal untuk atur jadwal fasting.", nil
	}

	user := &domain.User{
		Phone: phone,
		JID:   jid,
	}
	if err := u.userRepo.Create(user); err != nil {
		return "", fmt.Errorf("gagal mendaftar: %w", err)
	}

	return fmt.Sprintf("🎉 *Pendaftaran Berhasil!*\nNomor: %s\n\nSekarang atur jadwal fasting dengan:\n/jadwal HH:MM HH:MM\n\nContoh: /jadwal 05:00 18:00", phone), nil
}

func (u *fastingUsecase) SetSchedule(phone, start, end string) (string, error) {
	if _, err := time.Parse("15:04", start); err != nil {
		return "❌ Format waktu mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
	}
	if _, err := time.Parse("15:04", end); err != nil {
		return "❌ Format waktu selesai salah. Gunakan HH:MM (contoh: 18:00)", nil
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		return "❌ Kamu belum terdaftar. Kirim /daftar dulu.", nil
	}

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:    user.ID,
		FastStart: start,
		FastEnd:   end,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf("gagal menyimpan jadwal: %w", err)
	}

	return fmt.Sprintf("✅ *Jadwal Fasting Tersimpan!*\nMulai: %s\nSelesai: %s\n\nKamu akan menerima notifikasi otomatis.", start, end), nil
}

func (u *fastingUsecase) GetStatus(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		return "❌ Kamu belum terdaftar. Kirim /daftar dulu.", nil
	}

	name := user.Name
	if name == "" {
		name = user.Phone
	}

	schedule, err := u.scheduleRepo.FindActiveByUserID(user.ID)
	if err != nil {
		return "📋 *Status Fasting*\nBelum ada jadwal.\n\nAtur dengan: /jadwal HH:MM HH:MM", nil
	}

	now := time.Now()
	startTime, _ := time.Parse("15:04", schedule.FastStart)
	endTime, _ := time.Parse("15:04", schedule.FastEnd)

	startTime = time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
	endTime = time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, now.Location())

	var status string
	if now.Before(startTime) {
		status = fmt.Sprintf("⏳ Fasting dimulai dalam %s", formatDuration(startTime.Sub(now)))
	} else if now.Before(endTime) {
		status = fmt.Sprintf("🍽️ Sedang fasting! Sisa %s", formatDuration(endTime.Sub(now)))
	} else {
		status = "✅ Fasting hari ini sudah selesai!"
	}

	return fmt.Sprintf("📋 *Status Fasting*\nUser: %s\nJadwal: %s - %s\n\n%s", name, schedule.FastStart, schedule.FastEnd, status), nil
}

func (u *fastingUsecase) CancelToday(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		return "❌ Kamu belum terdaftar.", nil
	}

	if err := u.notificationRepo.LogNotification(user.ID, "cancelled"); err != nil {
		return "", fmt.Errorf("gagal membatalkan: %w", err)
	}

	return "✅ Fasting hari ini dibatalkan. Tidak akan ada notifikasi hari ini.", nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%d jam %d menit", hours, minutes)
	}
	return fmt.Sprintf("%d menit", minutes)
}
