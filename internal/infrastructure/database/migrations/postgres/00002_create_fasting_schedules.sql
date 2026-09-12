-- +goose Up
CREATE TABLE fasting_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    fast_start TEXT NOT NULL,
    fast_end TEXT NOT NULL,
    fasting_type_name TEXT DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE fasting_schedules;
