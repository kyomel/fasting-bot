package persistence

import (
	"time"

	"fasting-bot/internal/domain"
)

// Shared pure date/streak logic used by both the SQLite and PostgreSQL
// schedule repositories. Kept identical across dialects so streak semantics
// stay the same regardless of the backing database.

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
