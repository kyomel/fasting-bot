package persistence

import (
	"database/sql"
	"errors"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

const (
	storeDateTimeLayout = "2006-01-02 15:04"
	storeDateLayout     = "2006-01-02"
)

type ScheduleRepositorySQLite struct {
	db                      *sql.DB
	createStmt              *sql.Stmt
	deactivateByUserIDStmt  *sql.Stmt
	findActiveByUserIDStmt  *sql.Stmt
	createFastingRecordStmt *sql.Stmt
}

func NewScheduleRepository(db *sql.DB) repository.ScheduleRepository {
	r := &ScheduleRepositorySQLite{db: db}

	r.createStmt, _ = db.Prepare("INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name) VALUES (?, ?, ?, ?)")
	r.deactivateByUserIDStmt, _ = db.Prepare("UPDATE fasting_schedules SET is_active = 0 WHERE user_id = ?")
	r.findActiveByUserIDStmt, _ = db.Prepare("SELECT id, user_id, fast_start, fast_end, fasting_type_name, is_active, created_at FROM fasting_schedules WHERE user_id = ? AND is_active = 1 ORDER BY id DESC LIMIT 1")
	r.createFastingRecordStmt, _ = db.Prepare("INSERT INTO fasting_records (user_id, schedule_id, fasting_type_name, fast_start, planned_fast_end, opened_at, duration_minutes, completed_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")

	return r
}

func (r *ScheduleRepositorySQLite) Create(schedule *domain.FastingSchedule) error {
	result, err := r.createStmt.Exec(schedule.UserID, schedule.FastStart, schedule.FastEnd, schedule.FastingTypeName)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	schedule.ID = id
	return nil
}

func (r *ScheduleRepositorySQLite) DeactivateByUserID(userID int64) error {
	_, err := r.deactivateByUserIDStmt.Exec(userID)
	return err
}

func (r *ScheduleRepositorySQLite) FindActiveByUserID(userID int64) (*domain.FastingSchedule, error) {
	var schedule domain.FastingSchedule
	err := r.findActiveByUserIDStmt.QueryRow(userID).Scan(&schedule.ID, &schedule.UserID, &schedule.FastStart, &schedule.FastEnd, &schedule.FastingTypeName, &schedule.IsActive, &schedule.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *ScheduleRepositorySQLite) CreateFastingRecord(record *domain.FastingRecord) error {
	result, err := r.createFastingRecordStmt.Exec(record.UserID, record.ScheduleID, record.FastingTypeName, record.FastStart, record.PlannedFastEnd, record.OpenedAt, record.DurationMinutes, record.CompletedDate)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

func (r *ScheduleRepositorySQLite) UpsertFastingStats(record *domain.FastingRecord) error {
	fastStartDate, completedDate, fastingDays, err := fastingDateRange(record)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentStreakDays, longestStreakDays int
	var lastCompletedDate string
	err = tx.QueryRow(`
		SELECT current_streak_days, longest_streak_days, last_completed_date
		FROM user_fasting_stats
		WHERE user_id = ?
	`, record.UserID).Scan(&currentStreakDays, &longestStreakDays, &lastCompletedDate)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO user_fasting_stats (
				user_id,
				total_sessions,
				total_minutes,
				current_streak_days,
				longest_streak_days,
				last_completed_date,
				last_opened_at,
				last_duration_minutes,
				updated_at
			) VALUES (?, 1, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, record.UserID, record.DurationMinutes, fastingDays, fastingDays, record.CompletedDate, record.OpenedAt, record.DurationMinutes)
		if err != nil {
			return err
		}
		return tx.Commit()
	}

	nextCurrentStreakDays := nextCurrentStreakDays(currentStreakDays, lastCompletedDate, fastStartDate, completedDate, fastingDays)
	if nextCurrentStreakDays > longestStreakDays {
		longestStreakDays = nextCurrentStreakDays
	}

	_, err = tx.Exec(`
		UPDATE user_fasting_stats
		SET total_sessions = total_sessions + 1,
			total_minutes = total_minutes + ?,
			current_streak_days = ?,
			longest_streak_days = ?,
			last_completed_date = ?,
			last_opened_at = ?,
			last_duration_minutes = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, record.DurationMinutes, nextCurrentStreakDays, longestStreakDays, record.CompletedDate, record.OpenedAt, record.DurationMinutes, record.UserID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ScheduleRepositorySQLite) ResetStaleCurrentStreaks(currentDate, currentDateTime string) error {
	_, err := r.db.Exec(`
		UPDATE user_fasting_stats
		SET current_streak_days = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE current_streak_days > 0
		AND last_completed_date < date(?, '-1 day')
		AND NOT EXISTS (
			SELECT 1
			FROM fasting_schedules fs
			WHERE fs.user_id = user_fasting_stats.user_id
			AND fs.is_active = 1
			AND length(fs.fast_start) > 5
			AND length(fs.fast_end) > 5
			AND fs.fast_start <= ?
			AND date(fs.fast_start) <= date(user_fasting_stats.last_completed_date, '+1 day')
			AND date(fs.fast_end) >= date(user_fasting_stats.last_completed_date, '+1 day')
		)
	`, currentDate, currentDateTime)
	return err
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

func nextCurrentStreakDays(currentStreakDays int, lastCompletedDate string, fastStartDate, completedDate time.Time, fastingDays int) int {
	lastCompleted, err := time.Parse(storeDateLayout, lastCompletedDate)
	if err != nil {
		return fastingDays
	}

	if completedDate.Equal(lastCompleted) {
		if fastingDays > currentStreakDays {
			return fastingDays
		}
		return currentStreakDays
	}
	if completedDate.Before(lastCompleted) {
		return currentStreakDays
	}

	firstAllowedDate := lastCompleted.AddDate(0, 0, 1)
	if fastStartDate.After(firstAllowedDate) {
		return fastingDays
	}

	newCoveredDays := int(completedDate.Sub(lastCompleted).Hours() / 24)
	return currentStreakDays + newCoveredDays
}

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (r *ScheduleRepositorySQLite) FindFastingStatsByUserID(userID int64) (*domain.FastingStats, error) {
	var stats domain.FastingStats
	err := r.db.QueryRow(`
		SELECT s.user_id, COALESCE(NULLIF(u.name, ''), u.phone), s.total_sessions, s.total_minutes, s.current_streak_days, s.longest_streak_days, s.last_completed_date, s.last_opened_at, s.last_duration_minutes
		FROM user_fasting_stats s
		JOIN users u ON u.id = s.user_id
		WHERE s.user_id = ?
	`, userID).Scan(&stats.UserID, &stats.Name, &stats.TotalSessions, &stats.TotalMinutes, &stats.CurrentStreakDays, &stats.LongestStreakDays, &stats.LastCompletedDate, &stats.LastOpenedAt, &stats.LastDurationMinutes)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *ScheduleRepositorySQLite) FindFastingLeaderboard() ([]domain.FastingLeaderboardEntry, error) {
	rows, err := r.db.Query(`
		SELECT COALESCE(
			NULLIF(u.name, ''),
			CASE
				WHEN length(u.phone) >= 6 THEN substr(u.phone, 1, 3) || '***' || substr(u.phone, -2)
				ELSE 'Anon'
			END
		), s.current_streak_days, s.total_minutes, s.total_sessions
		FROM user_fasting_stats s
		JOIN users u ON u.id = s.user_id
		ORDER BY s.total_minutes DESC, s.current_streak_days DESC, s.total_sessions DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.FastingLeaderboardEntry
	for rows.Next() {
		var entry domain.FastingLeaderboardEntry
		if err := rows.Scan(&entry.Name, &entry.CurrentStreakDays, &entry.TotalMinutes, &entry.TotalSessions); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *ScheduleRepositorySQLite) CleanupOldFastingRecords(cutoff string) (int64, error) {
	result, err := r.db.Exec(`
		DELETE FROM fasting_records
		WHERE created_at < ?
		AND NOT EXISTS (
			SELECT 1 FROM fasting_schedules fs
			WHERE fs.user_id = fasting_records.user_id
			AND fs.is_active = 1
		)
	`, cutoff)
	if err != nil {
		return 0, err
	}
	deletedRecords, _ := result.RowsAffected()

	result, err = r.db.Exec(`
		DELETE FROM fasting_schedules
		WHERE is_active = 0
		AND created_at < ?
		AND NOT EXISTS (
			SELECT 1 FROM fasting_schedules active
			WHERE active.user_id = fasting_schedules.user_id
			AND active.is_active = 1
		)
	`, cutoff)
	if err != nil {
		return deletedRecords, err
	}
	deletedSchedules, _ := result.RowsAffected()
	if deletedRecords+deletedSchedules > 0 {
		_, _ = r.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	}

	return deletedRecords + deletedSchedules, nil
}

func (r *ScheduleRepositorySQLite) FindUsersToNotifyStart(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		WHERE fs.is_active = 1
		AND (
			(
				length(fs.fast_start) = 5
				AND fs.fast_start <= ?
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl
					WHERE nl.user_id = u.id
					AND nl.notification_type = 'start'
					AND DATE(nl.sent_at) = ?
				)
			)
			OR
			(
				length(fs.fast_start) > 5
				AND fs.fast_start <= ?
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl
					WHERE nl.user_id = u.id
					AND nl.notification_type = 'start'
					AND strftime('%Y-%m-%d %H:%M', nl.sent_at) >= fs.fast_start
				)
			)
		)
	`, currentTime, currentDate, currentDateTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func (r *ScheduleRepositorySQLite) FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		WHERE fs.is_active = 1
		AND (
			(
				length(fs.fast_end) = 5
				AND fs.fast_end <= ?
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl2
					WHERE nl2.user_id = u.id
					AND nl2.notification_type = 'end'
					AND DATE(nl2.sent_at) = ?
				)
			)
			OR
			(
				length(fs.fast_end) > 5
				AND fs.fast_end <= ?
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl2
					WHERE nl2.user_id = u.id
					AND nl2.notification_type = 'end'
					AND strftime('%Y-%m-%d %H:%M', nl2.sent_at) >= fs.fast_end
				)
			)
		)
	`, currentTime, currentDate, currentDateTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func scanNotificationTargets(rows *sql.Rows) ([]repository.NotificationTarget, error) {
	var targets []repository.NotificationTarget
	for rows.Next() {
		var t repository.NotificationTarget
		if err := rows.Scan(&t.UserID, &t.JID, &t.Phone, &t.Name, &t.FastStart, &t.FastEnd); err != nil {
			continue
		}
		targets = append(targets, t)
	}
	return targets, nil
}
