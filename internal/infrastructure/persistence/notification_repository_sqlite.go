package persistence

import (
	"database/sql"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/repository"
)

type NotificationRepositorySQLite struct {
	db           *sql.DB
	logNotifStmt *sql.Stmt
}

func NewNotificationRepository(db *sql.DB) repository.NotificationRepository {
	r := &NotificationRepositorySQLite{db: db}
	r.logNotifStmt, _ = db.Prepare("INSERT INTO notification_logs (user_id, notification_type, sent_at) VALUES (?, ?, ?)")
	return r
}

func (r *NotificationRepositorySQLite) LogNotification(userID int64, notificationType string) error {
	sentAt := time.Now().In(config.Location).Format("2006-01-02 15:04:05")
	_, err := r.logNotifStmt.Exec(userID, notificationType, sentAt)
	return err
}
