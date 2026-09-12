-- +goose Up
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_fasting_schedules_user_active ON fasting_schedules(user_id, is_active);
CREATE INDEX idx_fasting_schedules_active_start ON fasting_schedules(is_active, fast_start);
CREATE INDEX idx_fasting_schedules_active_end ON fasting_schedules(is_active, fast_end);
CREATE INDEX idx_fasting_schedules_inactive_created ON fasting_schedules(is_active, created_at);
CREATE INDEX idx_notification_logs_user ON notification_logs(user_id);
CREATE INDEX idx_notification_logs_user_type_sent ON notification_logs(user_id, notification_type, sent_at);
CREATE INDEX idx_fasting_records_user_date ON fasting_records(user_id, completed_date);
CREATE INDEX idx_fasting_records_total ON fasting_records(duration_minutes);
CREATE INDEX idx_fasting_records_created_at ON fasting_records(created_at);
CREATE INDEX idx_user_fasting_stats_total ON user_fasting_stats(total_minutes DESC);
CREATE INDEX idx_user_badges_user ON user_badges(user_id);

-- +goose Down
DROP INDEX idx_users_phone;
DROP INDEX idx_users_email;
DROP INDEX idx_users_username;
DROP INDEX idx_fasting_schedules_user_active;
DROP INDEX idx_fasting_schedules_active_start;
DROP INDEX idx_fasting_schedules_active_end;
DROP INDEX idx_fasting_schedules_inactive_created;
DROP INDEX idx_notification_logs_user;
DROP INDEX idx_notification_logs_user_type_sent;
DROP INDEX idx_fasting_records_user_date;
DROP INDEX idx_fasting_records_total;
DROP INDEX idx_fasting_records_created_at;
DROP INDEX idx_user_fasting_stats_total;
DROP INDEX idx_user_badges_user;
