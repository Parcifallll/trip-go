-- +goose Up
ALTER TABLE trips ADD COLUMN last_position_at TIMESTAMPTZ;

-- +goose Down
DROP COLUMN last_position_at;
