-- +goose Up
CREATE TABLE user_fasting_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    total_sessions INTEGER NOT NULL DEFAULT 0,
    total_minutes INTEGER NOT NULL DEFAULT 0,
    current_streak_days INTEGER NOT NULL DEFAULT 0,
    longest_streak_days INTEGER NOT NULL DEFAULT 0,
    last_completed_date TEXT NOT NULL DEFAULT '',
    last_streak_opened_at TEXT NOT NULL DEFAULT '',
    last_opened_at TEXT NOT NULL DEFAULT '',
    last_duration_minutes INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_fasting_stats;
