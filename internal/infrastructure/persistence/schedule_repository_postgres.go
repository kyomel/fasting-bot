package persistence

import (
	"database/sql"
	"errors"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type ScheduleRepositoryPostgres struct {
	db                      *sql.DB
	createStmt              *sql.Stmt
	deactivateByUserIDStmt  *sql.Stmt
	findActiveByUserIDStmt  *sql.Stmt
	createFastingRecordStmt *sql.Stmt
}

func NewScheduleRepositoryPostgres(db *sql.DB) repository.ScheduleRepository {
	r := &ScheduleRepositoryPostgres{db: db}

	r.createStmt, _ = db.Prepare("INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name) VALUES ($1, $2, $3, $4) RETURNING id")
	r.deactivateByUserIDStmt, _ = db.Prepare("UPDATE fasting_schedules SET is_active = false WHERE user_id = $1")
	r.findActiveByUserIDStmt, _ = db.Prepare("SELECT id, user_id, fast_start, fast_end, fasting_type_name, is_active, created_at FROM fasting_schedules WHERE user_id = $1 AND is_active = true ORDER BY id DESC LIMIT 1")
	r.createFastingRecordStmt, _ = db.Prepare("INSERT INTO fasting_records (user_id, schedule_id, fasting_type_name, fast_start, planned_fast_end, opened_at, duration_minutes, completed_date) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id")

	return r
}

func (r *ScheduleRepositoryPostgres) Create(schedule *domain.FastingSchedule) error {
	err := r.createStmt.QueryRow(string(schedule.UserID), schedule.FastStart, schedule.FastEnd, schedule.FastingTypeName).Scan(&schedule.ID)
	return err
}

func (r *ScheduleRepositoryPostgres) DeactivateByUserID(userID domain.ID) error {
	_, err := r.deactivateByUserIDStmt.Exec(string(userID))
	return err
}

func (r *ScheduleRepositoryPostgres) FindActiveByUserID(userID domain.ID) (*domain.FastingSchedule, error) {
	var schedule domain.FastingSchedule
	err := r.findActiveByUserIDStmt.QueryRow(string(userID)).Scan(&schedule.ID, &schedule.UserID, &schedule.FastStart, &schedule.FastEnd, &schedule.FastingTypeName, &schedule.IsActive, &schedule.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *ScheduleRepositoryPostgres) CreateFastingRecord(record *domain.FastingRecord) error {
	err := r.createFastingRecordStmt.QueryRow(
		string(record.UserID), string(record.ScheduleID), record.FastingTypeName,
		record.FastStart, record.PlannedFastEnd, record.OpenedAt,
		record.DurationMinutes, record.CompletedDate,
	).Scan(&record.ID)
	return err
}

func (r *ScheduleRepositoryPostgres) UpsertFastingStats(record *domain.FastingRecord) error {
	_, completedDate, _, err := fastingDateRange(record)
	if err != nil {
		return err
	}
	openedAt, err := time.Parse(storeDateTimeLayout, record.OpenedAt)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stats, err := r.getExistingStats(tx, record.UserID)
	if err != nil {
		return err
	}

	if stats == nil {
		return r.insertNewStats(tx, record, openedAt, completedDate)
	}

	return r.updateExistingStats(tx, record, openedAt, completedDate, stats)
}

func (r *ScheduleRepositoryPostgres) getExistingStats(tx *sql.Tx, userID domain.ID) (*userFastingStatsRow, error) {
	var stats userFastingStatsRow
	err := tx.QueryRow(`
		SELECT current_streak_days, longest_streak_days, last_completed_date, last_streak_opened_at, last_opened_at
		FROM user_fasting_stats
		WHERE user_id = $1
	`, string(userID)).Scan(&stats.currentStreakDays, &stats.longestStreakDays, &stats.lastCompletedDate, &stats.lastStreakOpenedAt, &stats.lastOpenedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *ScheduleRepositoryPostgres) insertNewStats(tx *sql.Tx, record *domain.FastingRecord, openedAt, completedDate time.Time) error {
	currentStreakDays := nextCurrentStreakDays(0, "", openedAt, record.StreakQualified)
	longestStreakDays := currentStreakDays
	lastCompletedDate := ""
	lastStreakOpenedAt := ""
	if record.StreakQualified {
		lastCompletedDate = completedDate.Format(storeDateLayout)
		lastStreakOpenedAt = record.OpenedAt
	}

	_, err := tx.Exec(`
		INSERT INTO user_fasting_stats (
			user_id,
			total_sessions,
			total_minutes,
			current_streak_days,
			longest_streak_days,
			last_completed_date,
			last_streak_opened_at,
			last_opened_at,
			last_duration_minutes,
			updated_at
		) VALUES ($1, 1, $2, $3, $4, $5, $6, $7, $8, now())
	`, string(record.UserID), record.DurationMinutes, currentStreakDays, longestStreakDays, lastCompletedDate, lastStreakOpenedAt, record.OpenedAt, record.DurationMinutes)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ScheduleRepositoryPostgres) updateExistingStats(tx *sql.Tx, record *domain.FastingRecord, openedAt, completedDate time.Time, stats *userFastingStatsRow) error {
	if stats.lastStreakOpenedAt == "" && stats.currentStreakDays > 0 {
		stats.lastStreakOpenedAt = stats.lastOpenedAt
	}

	nextCurrentStreakDays := nextCurrentStreakDays(stats.currentStreakDays, stats.lastStreakOpenedAt, openedAt, record.StreakQualified)
	if nextCurrentStreakDays > stats.longestStreakDays {
		stats.longestStreakDays = nextCurrentStreakDays
	}
	if record.StreakQualified && !openedAt.Before(parseStoredDateTimeOrZero(stats.lastStreakOpenedAt)) {
		stats.lastCompletedDate = completedDate.Format(storeDateLayout)
		stats.lastStreakOpenedAt = record.OpenedAt
	}

	_, err := tx.Exec(`
		UPDATE user_fasting_stats
		SET total_sessions = total_sessions + 1,
			total_minutes = total_minutes + $1,
			current_streak_days = $2,
			longest_streak_days = $3,
			last_completed_date = $4,
			last_streak_opened_at = $5,
			last_opened_at = $6,
			last_duration_minutes = $7,
			updated_at = now()
		WHERE user_id = $8
	`, record.DurationMinutes, nextCurrentStreakDays, stats.longestStreakDays, stats.lastCompletedDate, stats.lastStreakOpenedAt, record.OpenedAt, record.DurationMinutes, string(record.UserID))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ScheduleRepositoryPostgres) ResetStaleCurrentStreaks(currentDate, currentDateTime string) error {
	_ = currentDate
	_, err := r.db.Exec(`
		UPDATE user_fasting_stats
		SET current_streak_days = 0,
			updated_at = now()
		WHERE current_streak_days > 0
		AND last_streak_opened_at != ''
		AND (last_streak_opened_at::timestamptz + interval '24 hours') < $1::timestamptz
		AND NOT EXISTS (
			SELECT 1
			FROM fasting_schedules fs
			WHERE fs.user_id = user_fasting_stats.user_id
			AND fs.is_active = true
			AND char_length(fs.fast_start) > 5
			AND char_length(fs.fast_end) > 5
			AND fs.fast_start <= $2
		)
	`, currentDateTime, currentDateTime)
	return err
}

func (r *ScheduleRepositoryPostgres) FindFastingStatsByUserID(userID domain.ID) (*domain.FastingStats, error) {
	var stats domain.FastingStats
	err := r.db.QueryRow(`
		SELECT s.user_id, COALESCE(NULLIF(u.name, ''), u.phone), s.total_sessions, s.total_minutes, s.current_streak_days, s.longest_streak_days, s.last_completed_date, s.last_opened_at, s.last_duration_minutes
		FROM user_fasting_stats s
		JOIN users u ON u.id = s.user_id
		WHERE s.user_id = $1
	`, string(userID)).Scan(&stats.UserID, &stats.Name, &stats.TotalSessions, &stats.TotalMinutes, &stats.CurrentStreakDays, &stats.LongestStreakDays, &stats.LastCompletedDate, &stats.LastOpenedAt, &stats.LastDurationMinutes)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *ScheduleRepositoryPostgres) FindFastingLeaderboard() ([]domain.FastingLeaderboardEntry, error) {
	rows, err := r.db.Query(`
		SELECT u.id, COALESCE(
			NULLIF(u.name, ''),
			CASE
				WHEN char_length(u.phone) >= 6 THEN left(u.phone, 3) || '***' || right(u.phone, 2)
				ELSE 'Anon'
			END
		), s.current_streak_days, s.total_minutes, s.total_sessions
		FROM user_fasting_stats s
		JOIN users u ON u.id = s.user_id
		ORDER BY s.total_minutes DESC, s.current_streak_days DESC, s.total_sessions DESC, u.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.FastingLeaderboardEntry
	for rows.Next() {
		var entry domain.FastingLeaderboardEntry
		if err := rows.Scan(&entry.UserID, &entry.Name, &entry.CurrentStreakDays, &entry.TotalMinutes, &entry.TotalSessions); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *ScheduleRepositoryPostgres) FindRecentFastingRecords(userID domain.ID, limit int) ([]domain.FastingRecord, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, schedule_id, fasting_type_name, fast_start, planned_fast_end, opened_at, duration_minutes, completed_date, created_at
		FROM fasting_records
		WHERE user_id = $1
		ORDER BY opened_at DESC, id DESC
		LIMIT $2
	`, string(userID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.FastingRecord
	for rows.Next() {
		var record domain.FastingRecord
		if err := rows.Scan(
			&record.ID,
			&record.UserID,
			&record.ScheduleID,
			&record.FastingTypeName,
			&record.FastStart,
			&record.PlannedFastEnd,
			&record.OpenedAt,
			&record.DurationMinutes,
			&record.CompletedDate,
			&record.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *ScheduleRepositoryPostgres) CleanupOldFastingRecords(cutoff string) (int64, error) {
	result, err := r.db.Exec(`
		DELETE FROM fasting_records
		WHERE created_at < $1::timestamptz
		AND NOT EXISTS (
			SELECT 1 FROM fasting_schedules fs
			WHERE fs.user_id = fasting_records.user_id
			AND fs.is_active = true
		)
	`, cutoff)
	if err != nil {
		return 0, err
	}
	deletedRecords, _ := result.RowsAffected()

	result, err = r.db.Exec(`
		DELETE FROM fasting_schedules
		WHERE is_active = false
		AND created_at < $1::timestamptz
		AND NOT EXISTS (
			SELECT 1 FROM fasting_schedules active
			WHERE active.user_id = fasting_schedules.user_id
			AND active.is_active = true
		)
	`, cutoff)
	if err != nil {
		return deletedRecords, err
	}
	deletedSchedules, _ := result.RowsAffected()

	return deletedRecords + deletedSchedules, nil
}

func (r *ScheduleRepositoryPostgres) FindUsersToNotifyStart(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end, fs.fasting_type_name, COALESCE(ufs.current_streak_days, 0)
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		LEFT JOIN user_fasting_stats ufs ON u.id = ufs.user_id
		WHERE fs.is_active = true
		AND (
			(
				char_length(fs.fast_start) = 5
				AND fs.fast_start <= $1
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl
					WHERE nl.user_id = u.id
					AND nl.notification_type = 'start'
					AND nl.sent_at::date = $2::date
				)
			)
			OR
			(
				char_length(fs.fast_start) > 5
				AND fs.fast_start <= $3
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl
					WHERE nl.user_id = u.id
					AND nl.notification_type = 'start'
					AND to_char(nl.sent_at, 'YYYY-MM-DD HH24:MI') >= fs.fast_start
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

func (r *ScheduleRepositoryPostgres) FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end, fs.fasting_type_name, COALESCE(ufs.current_streak_days, 0)
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		LEFT JOIN user_fasting_stats ufs ON u.id = ufs.user_id
		WHERE fs.is_active = true
		AND (
			(
				char_length(fs.fast_end) = 5
				AND fs.fast_end <= $1
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl2
					WHERE nl2.user_id = u.id
					AND nl2.notification_type = 'end'
					AND nl2.sent_at::date = $2::date
				)
			)
			OR
			(
				char_length(fs.fast_end) > 5
				AND fs.fast_end <= $3
				AND NOT EXISTS (
					SELECT 1 FROM notification_logs nl2
					WHERE nl2.user_id = u.id
					AND nl2.notification_type = 'end'
					AND to_char(nl2.sent_at, 'YYYY-MM-DD HH24:MI') >= fs.fast_end
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

func (r *ScheduleRepositoryPostgres) FindUsersForElapsedNotification(notificationType string, triggerAfterHours int, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end, fs.fasting_type_name, COALESCE(ufs.current_streak_days, 0)
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		LEFT JOIN user_fasting_stats ufs ON u.id = ufs.user_id
		WHERE fs.is_active = true
		AND char_length(fs.fast_start) > 5
		AND char_length(fs.fast_end) > 5
		AND fs.fast_start::timestamptz <= $1::timestamptz
		AND fs.fast_end::timestamptz > $2::timestamptz
		AND (fs.fast_start::timestamptz + ($3::int * interval '1 hour')) <= $4::timestamptz
		AND (fs.fast_start::timestamptz + ($5::int * interval '1 hour') + interval '1 hour') > $6::timestamptz
		AND NOT EXISTS (
			SELECT 1 FROM notification_logs nl
			WHERE nl.user_id = u.id
			AND nl.notification_type = $7
			AND nl.sent_at::timestamptz >= fs.fast_start::timestamptz
		)
	`, currentDateTime, currentDateTime, triggerAfterHours, currentDateTime, triggerAfterHours, currentDateTime, notificationType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func (r *ScheduleRepositoryPostgres) FindUsersBeforeTargetNotification(notificationType string, leadHours int, currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end, fs.fasting_type_name, COALESCE(ufs.current_streak_days, 0)
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		LEFT JOIN user_fasting_stats ufs ON u.id = ufs.user_id
		WHERE fs.is_active = true
		AND char_length(fs.fast_start) > 5
		AND char_length(fs.fast_end) > 5
		AND fs.fast_start::timestamptz <= $1::timestamptz
		AND fs.fast_end::timestamptz > $2::timestamptz
		AND (fs.fast_end::timestamptz - ($3::int * interval '1 hour')) <= $4::timestamptz
		AND NOT EXISTS (
			SELECT 1 FROM notification_logs nl
			WHERE nl.user_id = u.id
			AND nl.notification_type IN ($5, 'near_target')
			AND nl.sent_at::timestamptz >= fs.fast_start::timestamptz
		)
	`, currentDateTime, currentDateTime, leadHours, currentDateTime, notificationType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func (r *ScheduleRepositoryPostgres) FindUsersWithActiveFasting(currentDateTime string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, COALESCE(NULLIF(u.name, ''), u.phone), fs.fast_start, fs.fast_end, fs.fasting_type_name, COALESCE(ufs.current_streak_days, 0)
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		LEFT JOIN user_fasting_stats ufs ON u.id = ufs.user_id
		WHERE fs.is_active = true
		AND fs.fast_start <= $1
		AND fs.fast_end > $2
	`, currentDateTime, currentDateTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func (r *ScheduleRepositoryPostgres) FindUsersWithExpiredStreaks(currentDateTime string) ([]repository.ExpiredStreakTarget, error) {
	rows, err := r.db.Query(`
		SELECT ufs.user_id, u.jid, COALESCE(NULLIF(u.name, ''), u.phone), ufs.current_streak_days
		FROM user_fasting_stats ufs
		JOIN users u ON u.id = ufs.user_id
		WHERE ufs.current_streak_days > 0
		AND ufs.last_streak_opened_at != ''
		AND (ufs.last_streak_opened_at::timestamptz + interval '24 hours') < $1::timestamptz
		AND NOT EXISTS (
			SELECT 1 FROM fasting_schedules fs
			WHERE fs.user_id = ufs.user_id
			AND fs.is_active = true
		)
	`, currentDateTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []repository.ExpiredStreakTarget
	for rows.Next() {
		var t repository.ExpiredStreakTarget
		if err := rows.Scan(&t.UserID, &t.JID, &t.Name, &t.CurrentStreakDays); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func (r *ScheduleRepositoryPostgres) ResetStreakByUserID(userID domain.ID) error {
	_, err := r.db.Exec(`
		UPDATE user_fasting_stats
		SET current_streak_days = 0, updated_at = now()
		WHERE user_id = $1
	`, string(userID))
	return err
}
