package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TrackEvent is one sighting of a vehicle by plate (extend later with DB, reports).
type TrackEvent struct {
	ID     int64     `json:"id"`
	Plate  string    `json:"plate"`
	Note   string    `json:"note,omitempty"`
	SeenAt time.Time `json:"seen_at"`
}

type createTrackRequest struct {
	Plate string `json:"plate"`
	Note  string `json:"note"`
}

// PlateCount is one row of the plate aggregation report.
type PlateCount struct {
	Plate string `json:"plate"`
	Count int64  `json:"count"`
}

type store struct {
	db *sql.DB
}

func openStore(dsn string) (*store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &store{db: db}, nil
}

func (s *store) Report() ([]PlateCount, error) {
	query := `SELECT plate, COUNT(*) AS cnt FROM track_events GROUP BY plate ORDER BY cnt DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PlateCount, 0)
	for rows.Next() {
		var r PlateCount
		if err := rows.Scan(&r.Plate, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// list returns track_events ordered by id. limit <= 0 means no row cap; offset is skipped rows (>= 0).
func (s *store) list(offset, limit int, plate string) ([]TrackEvent, error) {
	query := `SELECT id, plate, note, seen_at FROM track_events WHERE 1=1`
	var args []any
	if p := strings.TrimSpace(plate); p != "" {
		query += fmt.Sprintf(` AND plate ILIKE $%d`, len(args)+1)
		args = append(args, "%"+p+"%")
	}
	query += ` ORDER BY id`
	hasLimit := limit > 0
	hasOffset := offset > 0
	if hasLimit {
		query += fmt.Sprintf(` LIMIT $%d`, len(args)+1)
		args = append(args, limit)
	}
	if hasOffset {
		query += fmt.Sprintf(` OFFSET $%d`, len(args)+1)
		args = append(args, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TrackEvent, 0)
	for rows.Next() {
		var ev TrackEvent
		if err := rows.Scan(&ev.ID, &ev.Plate, &ev.Note, &ev.SeenAt); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *store) add(plate, note string) (TrackEvent, error) {
	now := time.Now().UTC()
	var id int64
	if err := s.db.QueryRow(
		`INSERT INTO track_events (plate, note, seen_at) VALUES ($1, $2, $3) RETURNING id`,
		plate, note, now,
	).Scan(&id); err != nil {
		return TrackEvent{}, err
	}
	return TrackEvent{
		ID:     id,
		Plate:  plate,
		Note:   note,
		SeenAt: now,
	}, nil
}

func (s *store) get(id int64) (TrackEvent, error) {
	var ev TrackEvent
	err := s.db.QueryRow(
		`SELECT id, plate, note, seen_at FROM track_events WHERE id = $1`,
		id,
	).Scan(&ev.ID, &ev.Plate, &ev.Note, &ev.SeenAt)
	if err != nil {
		return TrackEvent{}, err
	}
	return ev, nil
}

func (s *store) fixture() ([]TrackEvent, error) {
	list := []struct {
		plate string
		note  string
	}{
		{plate: "AB123C", note: "Some bloody wankers"},
		{plate: "AB234C", note: "Good lads"},
		{plate: "AB345C", note: "Fine gals"},
	}
	out := make([]TrackEvent, 0, len(list))
	for _, l := range list {
		ev, err := s.add(l.plate, l.note)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

// memory is set from main after openStore succeeds.
var memory *store
