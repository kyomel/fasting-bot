package persistence

import (
	"database/sql"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type NotificationRepositoryPostgres struct {
	db           *sql.DB
	logNotifStmt *sql.Stmt
}

func NewNotificationRepositoryPostgres(db *sql.DB) repository.NotificationRepository {
	r := &NotificationRepositoryPostgres{db: db}
	r.logNotifStmt, _ = db.Prepare("INSERT INTO notification_logs (user_id, notification_type, sent_at) VALUES ($1, $2, $3)")
	return r
}

func (r *NotificationRepositoryPostgres) LogNotification(userID domain.ID, notificationType string) error {
	sentAt := time.Now().In(config.Location).Format("2006-01-02 15:04:05")
	_, err := r.logNotifStmt.Exec(string(userID), notificationType, sentAt)
	return err
}
