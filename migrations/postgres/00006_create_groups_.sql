-- +goose Up
CREATE TABLE groups_ (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jid TEXT NOT NULL UNIQUE,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE groups_;
