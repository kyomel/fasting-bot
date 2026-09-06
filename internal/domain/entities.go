package domain

import "time"

type User struct {
	ID           ID
	Username     string
	PasswordHash string
	Phone        string
	Email        string
	Name         string
	JID          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type FastingSchedule struct {
	ID              ID
	UserID          ID
	FastStart       string
	FastEnd         string
	FastingTypeName string
	IsActive        bool
	CreatedAt       time.Time
}

type NotificationLog struct {
	ID               ID
	UserID           ID
	NotificationType string
	SentAt           time.Time
}

type FastingRecord struct {
	ID              ID
	UserID          ID
	ScheduleID      ID
	FastingTypeName string
	FastStart       string
	PlannedFastEnd  string
	OpenedAt        string
	DurationMinutes int
	CompletedDate   string
	StreakQualified bool
	CreatedAt       time.Time
}

type FastingStats struct {
	UserID              ID
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
	UserID            ID
	Name              string
	CurrentStreakDays int
	TotalMinutes      int
	TotalSessions     int
}

// NotificationTarget is a read-model returned by schedule queries to identify
// users that should receive a proactive notification.
type NotificationTarget struct {
	UserID            ID
	JID               string
	Phone             string
	Name              string
	FastStart         string
	FastEnd           string
	FastingTypeName   string
	CurrentStreakDays int
}

// ExpiredStreakTarget is a read-model for users whose streak has expired.
type ExpiredStreakTarget struct {
	UserID            ID
	JID               string
	Name              string
	CurrentStreakDays int
}
