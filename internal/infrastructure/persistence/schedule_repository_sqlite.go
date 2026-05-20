package persistence

import (
	"database/sql"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type ScheduleRepositorySQLite struct {
	db *sql.DB
}

func NewScheduleRepository(db *sql.DB) repository.ScheduleRepository {
	return &ScheduleRepositorySQLite{db: db}
}

func (r *ScheduleRepositorySQLite) Create(schedule *domain.FastingSchedule) error {
	result, err := r.db.Exec(
		"INSERT INTO fasting_schedules (user_id, fast_start, fast_end) VALUES (?, ?, ?)",
		schedule.UserID, schedule.FastStart, schedule.FastEnd,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	schedule.ID = id
	return nil
}

func (r *ScheduleRepositorySQLite) DeactivateByUserID(userID int64) error {
	_, err := r.db.Exec(
		"UPDATE fasting_schedules SET is_active = 0 WHERE user_id = ?",
		userID,
	)
	return err
}

func (r *ScheduleRepositorySQLite) FindActiveByUserID(userID int64) (*domain.FastingSchedule, error) {
	var schedule domain.FastingSchedule
	err := r.db.QueryRow(
		"SELECT id, user_id, fast_start, fast_end, is_active, created_at FROM fasting_schedules WHERE user_id = ? AND is_active = 1 ORDER BY id DESC LIMIT 1",
		userID,
	).Scan(&schedule.ID, &schedule.UserID, &schedule.FastStart, &schedule.FastEnd, &schedule.IsActive, &schedule.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *ScheduleRepositorySQLite) FindUsersToNotifyStart(currentTime, currentDate string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, fs.fast_start, fs.fast_end
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		WHERE fs.is_active = 1
		AND fs.fast_start = ?
		AND NOT EXISTS (
			SELECT 1 FROM notification_logs nl
			WHERE nl.user_id = u.id
			AND nl.notification_type = 'start'
			AND DATE(nl.sent_at) = ?
		)
	`, currentTime, currentDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanNotificationTargets(rows)
}

func (r *ScheduleRepositorySQLite) FindUsersToNotifyEnd(currentTime, currentDate string) ([]repository.NotificationTarget, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.jid, u.phone, fs.fast_start, fs.fast_end
		FROM users u
		JOIN fasting_schedules fs ON u.id = fs.user_id
		WHERE fs.is_active = 1
		AND fs.fast_end = ?
		AND EXISTS (
			SELECT 1 FROM notification_logs nl
			WHERE nl.user_id = u.id
			AND nl.notification_type = 'start'
			AND DATE(nl.sent_at) = ?
		)
		AND NOT EXISTS (
			SELECT 1 FROM notification_logs nl2
			WHERE nl2.user_id = u.id
			AND nl2.notification_type = 'end'
			AND DATE(nl2.sent_at) = ?
		)
	`, currentTime, currentDate, currentDate)
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
		if err := rows.Scan(&t.UserID, &t.JID, &t.Phone, &t.FastStart, &t.FastEnd); err != nil {
			continue
		}
		targets = append(targets, t)
	}
	return targets, nil
}
