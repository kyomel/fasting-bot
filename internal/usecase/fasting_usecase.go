package usecase

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

const (
	clockLayout       = "15:04"
	inputDateLayout   = "02-01-2006"
	storeLayout       = "2006-01-02 15:04"
	displayDateLayout = "02-01-2006 15:04"
)

type FastingUsecase interface {
	RegisterUser(phone, jid, name string) (string, error)
	SetName(phone, name string) (string, error)
	SetSchedule(phone, start, end string) (string, error)
	GetStatus(phone string) (string, error)
	CancelToday(phone string) (string, error)
	DeleteSchedule(phone string) (string, error)
	SetFastingType(phone string, typeID int, startTime string, durationHours int) (string, error)
	ScheduleFreestyleFasting(phone, kind, dateInput, startTime string, durationHours int) (string, error)
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

func (u *fastingUsecase) RegisterUser(phone, jid, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /daftar <nama>\nContoh: /daftar kyomel", nil
	}

	existingUser, err := u.userRepo.FindByPhone(phone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	if existingUser != nil && existingUser.ID > 0 {
		registeredName := existingUser.Name
		if registeredName == "" {
			registeredName = existingUser.Phone
		}
		return fmt.Sprintf("✅ Akun sudah terdaftar!\nID: %d\nNama: %s\nNomor: %s\n\nGunakan /setname <nama> untuk mengubah nama.", existingUser.ID, registeredName, existingUser.Phone), nil
	}

	user := &domain.User{
		Phone: phone,
		Name:  name,
		JID:   jid,
	}
	if err := u.userRepo.Create(user); err != nil {
		return "", fmt.Errorf("gagal mendaftar: %w", err)
	}

	return fmt.Sprintf("🎉 *Pendaftaran Berhasil!*\nID: %d\nNama: %s\nNomor: %s\n\nSekarang pilih jenis puasa:\n/list-puasa\n/set-puasa <nomor> <jam_mulai>\n\nContoh: /set-puasa 3 05:00", user.ID, name, phone), nil
}

func (u *fastingUsecase) SetName(phone, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /setname <nama baru>\nContoh: /setname kyomel baru", nil
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	if err := u.userRepo.UpdateName(user.ID, name); err != nil {
		return "", fmt.Errorf("gagal mengubah nama: %w", err)
	}

	return fmt.Sprintf("✅ Nama berhasil diubah menjadi: %s", name), nil
}

func (u *fastingUsecase) SetSchedule(phone, start, end string) (string, error) {
	startTime, err := nextStartFromClock(start)
	if err != nil {
		return "❌ Format waktu mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
	}
	endClock, err := parseClock(end)
	if err != nil {
		return "❌ Format waktu selesai salah. Gunakan HH:MM (contoh: 18:00)", nil
	}
	endTime := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), endClock.Hour(), endClock.Minute(), 0, 0, config.Location)
	if !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startTime),
		FastEnd:         formatStoredTime(endTime),
		FastingTypeName: "Manual",
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf("gagal menyimpan jadwal: %w", err)
	}

	return fmt.Sprintf("✅ *Jadwal Fasting Tersimpan!*\nMulai: %s\nSelesai: %s\n\nKamu akan menerima notifikasi otomatis.", formatDisplayTime(startTime), formatDisplayTime(endTime)), nil
}

func (u *fastingUsecase) GetStatus(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	name := user.Name
	if name == "" {
		name = user.Phone
	}

	schedule, err := u.scheduleRepo.FindActiveByUserID(user.ID)
	if err != nil {
		return fmt.Sprintf("📋 *Status Akun*\nID: %d\nNama: %s\nNomor: %s\n\nBelum ada jadwal fasting.\n\nAtur dengan: /list-puasa lalu /set-puasa <nomor> <jam_mulai>", user.ID, name, user.Phone), nil
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
		status = fmt.Sprintf("🍽️ Sedang fasting! Sisa %s", formatDuration(endTime.Sub(now)))
	} else {
		status = "✅ Fasting hari ini sudah selesai!"
	}

	return fmt.Sprintf("📋 *Status Fasting*\nID: %d\nNama: %s\nNomor: %s\nJenis Puasa: %s\nMulai: %s\nSelesai: %s\n\n%s", user.ID, name, user.Phone, fastingTypeName, formatScheduleDisplay(schedule.FastStart), formatScheduleDisplay(schedule.FastEnd), status), nil
}

func (u *fastingUsecase) CancelToday(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	if err := u.notificationRepo.LogNotification(user.ID, "cancelled"); err != nil {
		return "", fmt.Errorf("gagal membatalkan: %w", err)
	}

	return "✅ Fasting dibuka. Tidak akan ada notifikasi hari ini. Selamat berbuka! 🎉", nil
}

func (u *fastingUsecase) DeleteSchedule(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	if _, err := u.scheduleRepo.FindActiveByUserID(user.ID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gagal memeriksa jadwal: %w", err)
		}
		return "ℹ️ Belum ada jadwal fasting yang aktif untuk dihapus.", nil
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menghapus jadwal: %w", err)
	}

	return "✅ Jadwal fasting berhasil dihapus. Jika cek /status, jadwal tidak akan tampil lagi.", nil
}

func (u *fastingUsecase) SetFastingType(phone string, typeID int, startTime string, durationHours int) (string, error) {
	fastingType, err := domain.GetFastingTypeByID(typeID)
	if err != nil {
		return "❌ Jenis puasa tidak ditemukan. Kirim /list-puasa untuk melihat daftar.", nil
	}

	startDateTime, err := nextStartFromClock(startTime)
	if err != nil {
		return "❌ Format jam mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	var fastHours int
	var fastingTypeName string

	switch fastingType.ID {
	case 1, 2, 3, 4, 5, 6, 7:
		fastHours = fastingType.FastHours
		fastingTypeName = fastingType.Name
	case 8:
		if durationHours != 24 && durationHours != 36 && durationHours != 48 && durationHours != 72 {
			return "❌ Durasi Water Fasting harus 24, 36, 48, atau 72 jam.", nil
		}
		fastHours = durationHours
		fastingTypeName = fmt.Sprintf("Water Fasting %d jam", durationHours)
	case 9:
		if durationHours < 24 {
			return "❌ Water Fasting bebas minimal 24 jam.", nil
		}
		fastHours = durationHours
		fastingTypeName = fmt.Sprintf("Water Fasting %d jam", durationHours)
	case 10:
		if durationHours < 1 {
			return "❌ Durasi Dry Fasting harus minimal 1 jam.", nil
		}
		fastHours = durationHours
		fastingTypeName = fmt.Sprintf("Dry Fasting %d jam", durationHours)
	}
	endDateTime := calculateEndDateTime(startDateTime, fastHours)

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startDateTime),
		FastEnd:         formatStoredTime(endDateTime),
		FastingTypeName: fastingTypeName,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf("gagal menyimpan jadwal: %w", err)
	}

	return fmt.Sprintf("✅ *Jadwal %s Tersimpan!*\nMulai: %s\nSelesai: %s\n\nKamu akan menerima notifikasi otomatis.", fastingTypeName, formatDisplayTime(startDateTime), formatDisplayTime(endDateTime)), nil
}

func (u *fastingUsecase) ScheduleFreestyleFasting(phone, kind, dateInput, startTime string, durationHours int) (string, error) {
	fastingTypeName, err := freestyleFastingTypeName(kind)
	if err != nil {
		return "❌ Jenis puasa freestyle hanya WF atau DF.\nContoh: /jadwalkan WF 23-05-2026 16:00 12", nil
	}
	if durationHours < 1 {
		return "❌ Durasi puasa harus minimal 1 jam.", nil
	}

	startDateTime, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+startTime, config.Location)
	if err != nil {
		return "❌ Format jadwal salah. Gunakan: /jadwalkan WF DD-MM-YYYY HH:MM durasi_jam\nContoh: /jadwalkan WF 23-05-2026 16:00 12", nil
	}

	now := time.Now().In(config.Location)
	if !startDateTime.After(now) {
		return "❌ Tanggal dan jam mulai sudah lewat. Pilih waktu setelah sekarang.", nil
	}
	endDateTime := calculateEndDateTime(startDateTime, durationHours)

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu.", nil
		}
		return "", fmt.Errorf("gagal memeriksa data: %w", err)
	}

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startDateTime),
		FastEnd:         formatStoredTime(endDateTime),
		FastingTypeName: fastingTypeName,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf("gagal menyimpan jadwal: %w", err)
	}

	return fmt.Sprintf("✅ *Jadwal %s Freestyle Tersimpan!*\nMulai: %s\nSelesai: %s\nDurasi: %d jam\n\nKamu akan menerima notifikasi otomatis.", fastingTypeName, formatDisplayTime(startDateTime), formatDisplayTime(endDateTime), durationHours), nil
}

func freestyleFastingTypeName(kind string) (string, error) {
	switch kind {
	case "WF":
		return "Water Fasting (WF)", nil
	case "DF":
		return "Dry Fasting (DF)", nil
	default:
		return "", fmt.Errorf("jenis puasa freestyle tidak valid")
	}
}

func parseClock(value string) (time.Time, error) {
	return time.ParseInLocation(clockLayout, value, config.Location)
}

func nextStartFromClock(value string) (time.Time, error) {
	clock, err := parseClock(value)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(config.Location)
	nowMinute := now.Truncate(time.Minute)
	candidate := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, config.Location)
	if candidate.Before(nowMinute) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate, nil
}

func calculateEndDateTime(start time.Time, hours int) time.Time {
	return start.Add(time.Duration(hours) * time.Hour)
}

func formatStoredTime(t time.Time) string {
	return t.In(config.Location).Format(storeLayout)
}

func formatDisplayTime(t time.Time) string {
	return t.In(config.Location).Format(displayDateLayout)
}

func formatScheduleDisplay(value string) string {
	t, err := time.ParseInLocation(storeLayout, value, config.Location)
	if err != nil {
		return value
	}
	return formatDisplayTime(t)
}

func parseScheduleTime(value string, now time.Time) (time.Time, bool) {
	if t, err := time.ParseInLocation(storeLayout, value, config.Location); err == nil {
		return t, true
	}

	clock, err := parseClock(value)
	if err != nil {
		return now, false
	}
	return time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, config.Location), false
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%d jam %d menit", hours, minutes)
	}
	return fmt.Sprintf("%d menit", minutes)
}
