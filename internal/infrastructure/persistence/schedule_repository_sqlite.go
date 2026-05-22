package persistence

import (
	"database/sql"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
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
	_, err := r.db.Exec(`
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
		) VALUES (?, 1, ?, 1, 1, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			total_sessions = total_sessions + 1,
			total_minutes = total_minutes + excluded.total_minutes,
			current_streak_days = CASE
				WHEN user_fasting_stats.last_completed_date = excluded.last_completed_date THEN user_fasting_stats.current_streak_days
				WHEN user_fasting_stats.last_completed_date = date(excluded.last_completed_date, '-1 day') THEN user_fasting_stats.current_streak_days + 1
				ELSE 1
			END,
			longest_streak_days = MAX(
				user_fasting_stats.longest_streak_days,
				CASE
					WHEN user_fasting_stats.last_completed_date = excluded.last_completed_date THEN user_fasting_stats.current_streak_days
					WHEN user_fasting_stats.last_completed_date = date(excluded.last_completed_date, '-1 day') THEN user_fasting_stats.current_streak_days + 1
					ELSE 1
				END
			),
			last_completed_date = excluded.last_completed_date,
			last_opened_at = excluded.last_opened_at,
			last_duration_minutes = excluded.last_duration_minutes,
			updated_at = CURRENT_TIMESTAMP
	`, record.UserID, record.DurationMinutes, record.CompletedDate, record.OpenedAt, record.DurationMinutes)
	return err
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
		SELECT COALESCE(NULLIF(u.name, ''), u.phone), s.current_streak_days, s.total_minutes, s.total_sessions
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
