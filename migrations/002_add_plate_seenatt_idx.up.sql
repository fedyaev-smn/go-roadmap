CREATE INDEX IF NOT EXISTS track_events_plate_seen_at_idx
    ON track_events (plate, seen_at);