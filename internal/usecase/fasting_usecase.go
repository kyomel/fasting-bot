package usecase

import (
	"errors"
	"fmt"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/repository"
)

const (
	clockLayout       = "15:04"
	inputDateLayout   = "02-01-2006"
	storeLayout       = "2006-01-02 15:04"
	displayDateLayout = "02-01-2006 15:04"
)

const errCheckDataFormat = "gagal memeriksa data: %w"

// ErrValidation marks input-validation failures so the HTTP layer can map
// them to 400 without string matching.
var ErrValidation = errors.New("validation failed")

const msgNotRegistered = "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu."
const errSaveScheduleFormat = "gagal menyimpan jadwal: %w"

type FastingUsecase interface {
	RegisterUser(phone, jid, name string) (string, error)
	RegisterUserAPI(input RegisterInput) (*RegisterResult, error)
	SetName(phone, name string) (string, error)
	SetSchedule(phone, start, end string) (string, error)
	GetStatus(phone string) (string, error)
	CancelToday(phone string) (string, error)
	BreakFastingAt(phone, dateInput, openTime string) (string, error)
	DeleteSchedule(phone string) (string, error)
	GetStats(phone string) (string, error)
	GetHistory(phone string, limit int) (string, error)
	GetBadges(phone string) (string, error)
	GetLeaderboard() (string, error)
	GetMotivation(phone string) (string, error)
	GetPhases(phone string) (string, error)
	SetFastingByDuration(phone string, durationHours int, isDry bool, startTime string) (string, error)
	ScheduleFastingByDuration(phone string, durationHours int, isDry bool, dateInput, startTime string) (string, error)
}

type fastingUsecase struct {
	userRepo         repository.UserRepository
	scheduleRepo     repository.ScheduleRepository
	notificationRepo repository.NotificationRepository
	badgeRepo        repository.BadgeRepository
}

func NewFastingUsecase(
	userRepo repository.UserRepository,
	scheduleRepo repository.ScheduleRepository,
	notificationRepo repository.NotificationRepository,
	badgeRepo repository.BadgeRepository,
) FastingUsecase {
	return &fastingUsecase{
		userRepo:         userRepo,
		scheduleRepo:     scheduleRepo,
		notificationRepo: notificationRepo,
		badgeRepo:        badgeRepo,
	}
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
	totalMinutes := int(d.Minutes())
	totalHours := totalMinutes / 60
	minutes := totalMinutes % 60
	days := totalHours / 24
	hours := totalHours % 24

	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit (total: %d jam %d menit)", days, hours, minutes, totalHours, minutes)
	}
	if totalHours > 0 {
		return fmt.Sprintf("%d jam %d menit", totalHours, minutes)
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
	totalHours := totalMinutes / 60

	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit (total: %d jam %d menit)", days, hours, minutes, totalHours, minutes)
	}
	return fmt.Sprintf("%d jam %d menit", hours, minutes)
}

func displayFastingTypeName(name string) string {
	if name == "" {
		return "Belum diketahui"
	}
	return name
}
