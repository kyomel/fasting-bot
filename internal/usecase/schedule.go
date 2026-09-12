package usecase

import (
	"errors"
	"fmt"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

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
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menonaktifkan jadwal lama: %w", err)
	}

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

func (u *fastingUsecase) DeleteSchedule(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if _, err := u.scheduleRepo.FindActiveByUserID(user.ID); err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return "", fmt.Errorf("gagal memeriksa jadwal: %w", err)
		}
		return "ℹ️ Belum ada jadwal fasting yang aktif untuk dihapus.", nil
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menghapus jadwal: %w", err)
	}

	return "✅ Jadwal fasting berhasil dihapus. Jika cek /status, jadwal tidak akan tampil lagi.", nil
}

func (u *fastingUsecase) SetFastingByDuration(phone string, durationHours int, isDry bool, startTime string) (string, error) {
	if durationHours < 1 || durationHours > 168 {
		return "❌ Durasi harus 1-168 jam.", nil
	}
	if isDry && durationHours > 48 {
		return "❌ Dry fasting maksimal 48 jam dulu karena terlalu ekstrem.", nil
	}

	var startDateTime time.Time
	var err error
	if startTime == "" {
		startDateTime = time.Now().In(config.Location).Truncate(time.Minute)
	} else {
		startDateTime, err = nextStartFromClock(startTime)
		if err != nil {
			return "❌ Format jam mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
		}
	}

	fastingTypeName := buildFastingTypeName(durationHours, isDry)
	return u.saveFastingScheduleCore(phone, startDateTime, durationHours, fastingTypeName, false)
}

func (u *fastingUsecase) ScheduleFastingByDuration(phone string, durationHours int, isDry bool, dateInput, startTime string) (string, error) {
	if durationHours < 1 || durationHours > 168 {
		return "❌ Durasi harus 1-168 jam.", nil
	}
	if isDry && durationHours > 48 {
		return "❌ Dry fasting maksimal 48 jam dulu karena terlalu ekstrem.", nil
	}

	startDateTime, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+startTime, config.Location)
	if err != nil {
		return "❌ Format jadwal salah. Gunakan: /jadwal <durasi> DD-MM-YYYY HH:MM\nContoh: /jadwal 16 20-06-2026 05:00", nil
	}

	fastingTypeName := buildFastingTypeName(durationHours, isDry)
	return u.saveFastingScheduleCore(phone, startDateTime, durationHours, fastingTypeName, true)
}

func buildFastingTypeName(durationHours int, isDry bool) string {
	if isDry {
		return fmt.Sprintf("Dry Fasting %d jam", durationHours)
	}
	if durationHours >= 24 {
		return fmt.Sprintf("Water Fasting %d jam", durationHours)
	}
	if durationHours >= 22 {
		return fmt.Sprintf("OMAD %d jam", durationHours)
	}
	return fmt.Sprintf("IF %d jam", durationHours)
}

func (u *fastingUsecase) saveFastingScheduleCore(phone string, startDateTime time.Time, durationHours int, fastingTypeName string, markElapsedNotifications bool) (string, error) {
	endDateTime := startDateTime.Add(time.Duration(durationHours) * time.Hour)

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menonaktifkan jadwal lama: %w", err)
	}

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startDateTime),
		FastEnd:         formatStoredTime(endDateTime),
		FastingTypeName: fastingTypeName,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf(errSaveScheduleFormat, err)
	}
	if markElapsedNotifications {
		u.markElapsedNotifications(user.ID, startDateTime, endDateTime)
	}

	return fmt.Sprintf(
		"🎯 *Jadwal %s tersimpan!*\n"+
			"⏱ Mulai: *%s*\n"+
			"🏁 Buka: *%s* (%d jam)\n\n"+
			"%s\n\n"+
			"Kamu akan dapat notifikasi otomatis saat mulai & saat waktunya buka.\n"+
			"_Mau ganti? Jalankan /puasa lagi — jadwal lama otomatis dimatikan._",
		fastingTypeName, formatDisplayTime(startDateTime), formatDisplayTime(endDateTime),
		durationHours, scheduleTeaserForDuration(durationHours, fastingTypeName),
	), nil
}

// scheduleTeaserForDuration shows a 1-line, secular preview of what the body
// will do during the planned fast — appears in the schedule-saved confirmation.
func scheduleTeaserForDuration(durationHours int, fastingTypeName string) string {
	if isDryFasting(fastingTypeName) {
		return "💡 Dry fasting butuh ekstra sadar sinyal tubuh: pantau pusing, lemas berlebihan, dan suhu tubuh. Kalau tubuh memberi sinyal keras, batalkan dengan bijak."
	}

	switch {
	case durationHours <= 0:
		return "Tubuhmu akan mulai turunkan insulin & masuk mode repair."
	case durationHours <= 14:
		return "💡 Di rentang ini: insulin turun, glikogen mulai dipakai, sel mulai shift ke mode \"akses\" lemak."
	case durationHours <= 20:
		return "💡 Di rentang ini: *metabolic switch* aktif, ketogenesis dimulai, autophagy ramp-up — Yoshinori Ohsumi dapat Nobel buat proses ini."
	case durationHours <= 36:
		return "💡 Di rentang ini: HGH spike (proteksi otot), IGF-1 turun, autophagy puncak. Wajib hidrasi + sedikit garam."
	case durationHours <= 72:
		return "💡 Di rentang ini: sinyal regenerasi stem cell aktif (Valter Longo, Cell 2014). ⚠️ Elektrolit wajib: Na, K, Mg tiap hari."
	default:
		return "💡 Zona prolonged fasting (>72 jam): imun reset & deep cellular cleanup. ⚠️ Wajib hidrasi + elektrolit, refeed pelan-pelan, konsultasi dokter kalau ada kondisi medis."
	}
}

func (u *fastingUsecase) markElapsedNotifications(userID domain.ID, startDateTime, endDateTime time.Time) {
	now := time.Now().In(config.Location).Truncate(time.Minute)
	if !startDateTime.After(now) {
		_ = u.notificationRepo.LogNotification(userID, "start")
	}
	if !endDateTime.After(now) {
		_ = u.notificationRepo.LogNotification(userID, "end")
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
