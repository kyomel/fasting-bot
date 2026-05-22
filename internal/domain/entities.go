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
	ID              int64
	UserID          int64
	FastStart       string
	FastEnd         string
	FastingTypeName string
	IsActive        bool
	CreatedAt       time.Time
}

type NotificationLog struct {
	ID               int64
	UserID           int64
	NotificationType string
	SentAt           time.Time
}

type FastingRecord struct {
	ID              int64
	UserID          int64
	ScheduleID      int64
	FastingTypeName string
	FastStart       string
	PlannedFastEnd  string
	OpenedAt        string
	DurationMinutes int
	CompletedDate   string
	CreatedAt       time.Time
}

type FastingStats struct {
	UserID              int64
	Name                string
	TotalSessions       int
	TotalMinutes        int
	CurrentStreakDays   int
	LongestStreakDays   int
	LastCompletedDate   string
	LastOpenedAt        string
	LastDurationMinutes int
}

type FastingLeaderboardEntry struct {
	Name              string
	CurrentStreakDays int
	TotalMinutes      int
	TotalSessions     int
}
