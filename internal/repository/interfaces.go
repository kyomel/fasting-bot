package repository

import (
	"fasting-bot/internal/domain"
)

type UserRepository interface {
	Create(user *domain.User) error
	FindByPhone(phone string) (*domain.User, error)
	FindByID(id int64) (*domain.User, error)
}

type ScheduleRepository interface {
	Create(schedule *domain.FastingSchedule) error
	DeactivateByUserID(userID int64) error
	FindActiveByUserID(userID int64) (*domain.FastingSchedule, error)
	FindUsersToNotifyStart(currentTime, currentDate string) ([]NotificationTarget, error)
	FindUsersToNotifyEnd(currentTime, currentDate string) ([]NotificationTarget, error)
}

type NotificationRepository interface {
	LogNotification(userID int64, notificationType string) error
}

type NotificationTarget struct {
	UserID    int64
	JID       string
	Phone     string
	FastStart string
	FastEnd   string
}
