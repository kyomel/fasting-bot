package persistence

import (
	"database/sql"
	"fasting-bot/internal/repository"
)

type NotificationRepositorySQLite struct {
	db          *sql.DB
	logNotifStmt *sql.Stmt
}

func NewNotificationRepository(db *sql.DB) repository.NotificationRepository {
	r := &NotificationRepositorySQLite{db: db}
	r.logNotifStmt, _ = db.Prepare("INSERT INTO notification_logs (user_id, notification_type) VALUES (?, ?)")
	return r
}

func (r *NotificationRepositorySQLite) LogNotification(userID int64, notificationType string) error {
	_, err := r.logNotifStmt.Exec(userID, notificationType)
	return err
}