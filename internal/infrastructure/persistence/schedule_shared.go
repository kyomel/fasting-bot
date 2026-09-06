package persistence

import (
	"database/sql"
	"time"

	"fasting-bot/internal/domain"
)

// Shared date/streak logic and row helpers for the schedule repository.
const (
	storeDateTimeLayout = "2006-01-02 15:04"
	storeDateLayout     = "2006-01-02"
)

type userFastingStatsRow struct {
	currentStreakDays  int
	longestStreakDays  int
	lastCompletedDate  string
	lastStreakOpenedAt string
	lastOpenedAt       string
}

func scanNotificationTargets(rows *sql.Rows) ([]domain.NotificationTarget, error) {
	var targets []domain.NotificationTarget
	for rows.Next() {
		var t domain.NotificationTarget
		if err := rows.Scan(&t.UserID, &t.JID, &t.Phone, &t.Name, &t.FastStart, &t.FastEnd, &t.FastingTypeName, &t.CurrentStreakDays); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func fastingDateRange(record *domain.FastingRecord) (time.Time, time.Time, int, error) {
	completedDate, err := time.Parse(storeDateLayout, record.CompletedDate)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	fastStart, err := time.Parse(storeDateTimeLayout, record.FastStart)
	if err != nil {
		fastStart = completedDate
	}
	fastStartDate := truncateDate(fastStart)
	if completedDate.Before(fastStartDate) {
		completedDate = fastStartDate
	}

	fastingDays := int(completedDate.Sub(fastStartDate).Hours()/24) + 1
	return fastStartDate, completedDate, fastingDays, nil
}

func nextCurrentStreakDays(currentStreakDays int, lastStreakOpenedAt string, openedAt time.Time, streakQualified bool) int {
	if !streakQualified {
		if isStreakExpired(lastStreakOpenedAt, openedAt) {
			return 0
		}
		return currentStreakDays
	}

	lastOpened, err := time.Parse(storeDateTimeLayout, lastStreakOpenedAt)
	if err != nil {
		return 1
	}
	if openedAt.Before(lastOpened) {
		return currentStreakDays
	}
	if openedAt.Sub(lastOpened) > 24*time.Hour {
		return 1
	}

	return currentStreakDays + 1
}

func isStreakExpired(lastStreakOpenedAt string, now time.Time) bool {
	lastOpened, err := time.Parse(storeDateTimeLayout, lastStreakOpenedAt)
	if err != nil {
		return false
	}
	return now.Sub(lastOpened) > 24*time.Hour
}

func parseStoredDateTimeOrZero(value string) time.Time {
	t, err := time.Parse(storeDateTimeLayout, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
