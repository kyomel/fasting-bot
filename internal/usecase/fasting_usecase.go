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

const errCheckDataFormat = "gagal memeriksa data: %w"
const msgNotRegistered = "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu."
const errSaveScheduleFormat = "gagal menyimpan jadwal: %w"

type FastingUsecase interface {
	RegisterUser(phone, jid, name string) (string, error)
	SetName(phone, name string) (string, error)
	SetSchedule(phone, start, end string) (string, error)
	GetStatus(phone string) (string, error)
	CancelToday(phone string) (string, error)
	DeleteSchedule(phone string) (string, error)
	GetStats(phone string) (string, error)
	GetLeaderboard() (string, error)
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
		return "", fmt.Errorf(errCheckDataFormat, err)
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
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
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
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startTime),
		FastEnd:         formatStoredTime(endTime),
		FastingTypeName: "Manual",
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf(errSaveScheduleFormat, err)
	}

	return fmt.Sprintf("✅ *Jadwal Fasting Tersimpan!*\nMulai: %s\nSelesai: %s\n\nKamu akan menerima notifikasi otomatis.", formatDisplayTime(startTime), formatDisplayTime(endTime)), nil
}

func (u *fastingUsecase) GetStatus(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
		status = fmt.Sprintf("🍽️ Sedang fasting!\nSudah berjalan: %s\nSisa: %s", formatDuration(now.Sub(startTime)), formatDuration(endTime.Sub(now)))
	} else {
		status = "✅ Fasting hari ini sudah selesai!"
	}

	return fmt.Sprintf("📋 *Status Fasting*\nID: %d\nNama: %s\nNomor: %s\nJenis Puasa: %s\nMulai: %s\nSelesai: %s\n\n%s", user.ID, name, user.Phone, fastingTypeName, formatScheduleDisplay(schedule.FastStart), formatScheduleDisplay(schedule.FastEnd), status), nil
}

func (u *fastingUsecase) CancelToday(phone string) (string, error) {
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

	now := time.Now().In(config.Location)
	startTime, _ := parseScheduleTime(schedule.FastStart, now)
	if now.Before(startTime) {
		return u.cancelBeforeStart(user.ID, startTime)
	}
	return u.breakFasting(user, schedule, startTime, now)
}

func (u *fastingUsecase) lookupUser(phone string) (*domain.User, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}
	return user, nil
}

func (u *fastingUsecase) lookupActiveSchedule(userID int64) (*domain.FastingSchedule, error) {
	schedule, err := u.scheduleRepo.FindActiveByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal memeriksa jadwal: %w", err)
	}
	return schedule, nil
}

func (u *fastingUsecase) cancelBeforeStart(userID int64, startTime time.Time) (string, error) {
	if err := u.scheduleRepo.DeactivateByUserID(userID); err != nil {
		return "", fmt.Errorf("gagal membatalkan jadwal: %w", err)
	}
	if err := u.notificationRepo.LogNotification(userID, "cancelled"); err != nil {
		return "", fmt.Errorf("gagal mencatat pembatalan: %w", err)
	}
	return fmt.Sprintf("ℹ️ Jadwal fasting dibatalkan.\nMulai: %s\n\nKarena /buka dilakukan sebelum jam puasa mulai, durasi tidak dihitung ke /stats.", formatDisplayTime(startTime)), nil
}

func (u *fastingUsecase) breakFasting(user *domain.User, schedule *domain.FastingSchedule, startTime, now time.Time) (string, error) {
	durationMinutes := int(now.Sub(startTime).Minutes())
	if durationMinutes < 0 {
		durationMinutes = 0
	}

	record := &domain.FastingRecord{
		UserID:          user.ID,
		ScheduleID:      schedule.ID,
		FastingTypeName: schedule.FastingTypeName,
		FastStart:       schedule.FastStart,
		PlannedFastEnd:  schedule.FastEnd,
		OpenedAt:        formatStoredTime(now),
		DurationMinutes: durationMinutes,
		CompletedDate:   now.Format("2006-01-02"),
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

	return fmt.Sprintf("✅ Fasting dibuka. Selamat berbuka! 🎉\nJenis Puasa: %s\nMulai: %s\nBuka: %s\nTotal waktu puasa: %s\n\nHasil ini sudah masuk ke /stats.", displayFastingTypeName(schedule.FastingTypeName), formatDisplayTime(startTime), formatDisplayTime(now), formatDurationWithDays(durationMinutes)), nil
}

func (u *fastingUsecase) DeleteSchedule(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
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

func (u *fastingUsecase) GetStats(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}

	stats, err := u.scheduleRepo.FindFastingStatsByUserID(user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
		}
		return "", fmt.Errorf("gagal mengambil stats: %w", err)
	}
	if stats.TotalSessions == 0 {
		return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
	}

	return fmt.Sprintf("📊 *Stats Puasa %s*\nTotal sesi: %d\nStreak puasa saat ini: %d hari\nStreak puasa terpanjang: %d hari\nTotal waktu puasa: %s\n\nTerakhir buka: %s\nDurasi terakhir: %s", stats.Name, stats.TotalSessions, stats.CurrentStreakDays, stats.LongestStreakDays, formatDurationWithDays(stats.TotalMinutes), formatScheduleDisplay(stats.LastOpenedAt), formatDurationWithDays(stats.LastDurationMinutes)), nil
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
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
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
		if durationHours < 1 {
			return "❌ Durasi Dry Fasting harus minimal 1 jam.", nil
		}
		fastHours = durationHours
		fastingTypeName = fmt.Sprintf("Dry Fasting %d jam", durationHours)
	case 10:
		if durationHours < 24 {
			return "❌ Prolonged Fasting metode water fasting minimal 24 jam.", nil
		}
		fastHours = durationHours
		fastingTypeName = fmt.Sprintf("Prolonged Fasting (Water) %d jam", durationHours)
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
		return "", fmt.Errorf(errSaveScheduleFormat, err)
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
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	u.scheduleRepo.DeactivateByUserID(user.ID)

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startDateTime),
		FastEnd:         formatStoredTime(endDateTime),
		FastingTypeName: fastingTypeName,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf(errSaveScheduleFormat, err)
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

func (u *fastingUsecase) refreshStaleCurrentStreaks() error {
	now := time.Now().In(config.Location)
	return u.scheduleRepo.ResetStaleCurrentStreaks(now.Format("2006-01-02"), formatStoredTime(now))
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

func formatDurationWithDays(totalMinutes int) string {
	if totalMinutes < 0 {
		totalMinutes = 0
	}

	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60
	return fmt.Sprintf("%d hari %d jam %d menit", days, hours, minutes)
}

func displayFastingTypeName(name string) string {
	if name == "" {
		return "Belum diketahui"
	}
	return name
}
