package persistence

import (
	"database/sql"
	"fasting-bot/internal/repository"
)

type NotificationRepositorySQLite struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) repository.NotificationRepository {
	return &NotificationRepositorySQLite{db: db}
}

func (r *NotificationRepositorySQLite) LogNotification(userID int64, notificationType string) error {
	_, err := r.db.Exec(
		"INSERT INTO notification_logs (user_id, notification_type) VALUES (?, ?)",
		userID, notificationType,
	)
	return err
}
