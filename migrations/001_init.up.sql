CREATE TABLE IF NOT EXISTS track_events
(
    id BIGSERIAL PRIMARY KEY,
    plate TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    seen_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS track_events_plate_idx ON track_events (plate);