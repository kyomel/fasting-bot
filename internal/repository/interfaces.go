package repository

import (
	"fasting-bot/internal/domain"
)

type UserRepository interface {
	Create(user *domain.User) error
	UpdateName(userID int64, name string) error
	FindByPhone(phone string) (*domain.User, error)
	FindByID(id int64) (*domain.User, error)
}

type ScheduleRepository interface {
	Create(schedule *domain.FastingSchedule) error
	DeactivateByUserID(userID int64) error
	FindActiveByUserID(userID int64) (*domain.FastingSchedule, error)
	CreateFastingRecord(record *domain.FastingRecord) error
	UpsertFastingStats(record *domain.FastingRecord) error
	FindFastingStatsByUserID(userID int64) (*domain.FastingStats, error)
	FindFastingLeaderboard() ([]domain.FastingLeaderboardEntry, error)
	CleanupOldFastingRecords(cutoff string) (int64, error)
	FindUsersToNotifyStart(currentTime, currentDate, currentDateTime string) ([]NotificationTarget, error)
	FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime string) ([]NotificationTarget, error)
}

type NotificationRepository interface {
	LogNotification(userID int64, notificationType string) error
}

type NotificationTarget struct {
	UserID    int64
	JID       string
	Phone     string
	Name      string
	FastStart string
	FastEnd   string
}
