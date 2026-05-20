package domain

import "time"

type User struct {
	ID        int64
	Phone     string
	Name      string
	JID       string
	CreatedAt time.Time
}

type FastingSchedule struct {
	ID        int64
	UserID    int64
	FastStart string
	FastEnd   string
	IsActive  bool
	CreatedAt time.Time
}

type NotificationLog struct {
	ID               int64
	UserID           int64
	NotificationType string
	SentAt           time.Time
}
