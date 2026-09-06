package repository

import (
	"fasting-bot/internal/domain"
)

type UserRepository interface {
	Create(user *domain.User) error
	UpdateName(userID domain.ID, name string) error
	FindByPhone(phone string) (*domain.User, error)
	FindByUsername(username string) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	FindByID(id domain.ID) (*domain.User, error)
}

type ScheduleRepository interface {
	Create(schedule *domain.FastingSchedule) error
	DeactivateByUserID(userID domain.ID) error
	FindActiveByUserID(userID domain.ID) (*domain.FastingSchedule, error)
	CreateFastingRecord(record *domain.FastingRecord) error
	UpsertFastingStats(record *domain.FastingRecord) error
	ResetStaleCurrentStreaks(currentDate, currentDateTime string) error
	FindFastingStatsByUserID(userID domain.ID) (*domain.FastingStats, error)
	FindRecentFastingRecords(userID domain.ID, limit int) ([]domain.FastingRecord, error)
	FindFastingLeaderboard() ([]domain.FastingLeaderboardEntry, error)
	CleanupOldFastingRecords(cutoff string) (int64, error)
	FindUsersToNotifyStart(currentTime, currentDate, currentDateTime string) ([]domain.NotificationTarget, error)
	FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime string) ([]domain.NotificationTarget, error)
	FindUsersForElapsedNotification(notificationType string, triggerAfterHours int, currentDateTime string) ([]domain.NotificationTarget, error)
	FindUsersBeforeTargetNotification(notificationType string, leadHours int, currentDateTime string) ([]domain.NotificationTarget, error)
	FindUsersWithActiveFasting(currentDateTime string) ([]domain.NotificationTarget, error)
	FindUsersWithExpiredStreaks(currentDateTime string) ([]domain.ExpiredStreakTarget, error)
	ResetStreakByUserID(userID domain.ID) error
}

type NotificationRepository interface {
	LogNotification(userID domain.ID, notificationType string) error
}

type BadgeRepository interface {
	EarnedBadges(userID domain.ID) (map[domain.BadgeKey]struct{}, error)
	AwardBadges(userID domain.ID, keys []domain.BadgeKey) error
}
