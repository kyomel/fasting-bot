-- +goose Up
CREATE TABLE fasting_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    schedule_id UUID NOT NULL,
    fasting_type_name TEXT DEFAULT '',
    fast_start TEXT NOT NULL,
    planned_fast_end TEXT NOT NULL,
    opened_at TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL,
    completed_date TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE fasting_records;
