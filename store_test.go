package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStoreGet(t *testing.T) {
	st := getSt(t)

	// Insert a row
	wantPlate := "TEST123"
	wantNote := "get test"
	wantSeen := time.Now().UTC().Truncate(time.Microsecond)
	id := insertRow(t, st, wantPlate, wantNote, wantSeen)
	t.Cleanup(func() {
		_, _ = st.db.Exec(`DELETE FROM track_events WHERE id = $1`, id)
	})

	got, err := st.get(id)
	if err != nil {
		t.Fatal(err)
	}

	// Case unexpected ID
	if got.ID != id {
		t.Fatalf("ID: got %d want %d", got.ID, id)
	}

	// Case unexpected Plate
	if got.Plate != wantPlate {
		t.Fatalf("Plate: got %q want %q", got.Plate, wantPlate)
	}

	// Case unexpected Note
	if got.Note != wantNote {
		t.Fatalf("Note: got %q want %q", got.Note, wantNote)
	}

	// Case unexpected SeenAt
	d := got.SeenAt.Sub(wantSeen)
	if d < 0 {
		d = -d
	}
	if d > time.Millisecond {
		t.Fatalf("SeenAt delta too big: %s", d)
	}

	// Case not found
	_, err = st.get(id + 999999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestStoreDelete(t *testing.T) {
	st := getSt(t)

	// Insert a row
	id := insertRow(t, st, "TEST123", "get test", time.Now().UTC().Truncate(time.Microsecond))
	t.Cleanup(func() {
		_, _ = st.db.Exec(`DELETE FROM track_events WHERE id = $1`, id)
	})

	// Case delete positive
	if err := st.delete(id); err != nil {
		t.Fatal(err)
	}
	// now GET on same id should be not found
	_, err := st.get(id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}

	// Case delete negative
	err = st.delete(id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows on second delete, got %v", err)
	}
}

func TestStoreReport(t *testing.T) {
	st := getSt(t)

	// Unique prefix so we can filter results without touching real data.
	prefix := fmt.Sprintf("TEST_PLATE_%d_", time.Now().UnixNano())
	seenAtDef := time.Now().UTC().Truncate(time.Microsecond)

	cases := []struct {
		plate  string
		note   string
		seenAt time.Time
	}{
		{plate: prefix + "AA111", note: "x", seenAt: seenAtDef.AddDate(0, 0, -1)},
		{plate: prefix + "AA111", note: "y", seenAt: seenAtDef},
		{plate: prefix + "BB222", note: "z", seenAt: seenAtDef.AddDate(0, 0, 1)},
	}

	// Insert rows
	ids := make([]int64, 0, len(cases))
	for _, c := range cases {
		id := insertRow(t, st, c.plate, c.note, c.seenAt)
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = st.db.Exec(`DELETE FROM track_events WHERE id = $1`, id)
		}
	})

	// Case no filters
	res, err := st.Report("", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// Filter report rows down to only our test plates.
	onlyTest := make([]PlateCount, 0)
	for _, r := range res {
		if strings.HasPrefix(r.Plate, prefix) {
			onlyTest = append(onlyTest, r)
		}
	}
	expected := []PlateCount{
		{Plate: prefix + "AA111", Count: 2},
		{Plate: prefix + "BB222", Count: 1},
	}
	if !reflect.DeepEqual(onlyTest, expected) {
		t.Fatalf("got %#v want %#v", onlyTest, expected)
	}

	// Case filter plate
	wantPlate := prefix + "AA111"
	res, err = st.Report(wantPlate, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	expected = []PlateCount{
		{Plate: prefix + "AA111", Count: 2},
	}
	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("got %#v want %#v", res, expected)
	}

	// Case filter from to
	wantFrom := seenAtDef
	wantTo := seenAtDef.AddDate(0, 0, 2)
	res, err = st.Report(prefix, wantFrom, wantTo)
	if err != nil {
		t.Fatal(err)
	}
	expected = []PlateCount{
		{Plate: prefix + "AA111", Count: 1},
		{Plate: prefix + "BB222", Count: 1},
	}
	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("got %#v want %#v", res, expected)
	}
}

func TestStoreList(t *testing.T) {
	st := getSt(t)

	seenAt := time.Now().UTC().Truncate(time.Microsecond)
	prefix := fmt.Sprintf("TEST_PLATE_%d_", time.Now().UnixNano())

	rows := []struct {
		plate  string
		note   string
		seenAt time.Time
	}{
		{prefix + "TEST_PLATE_1", "q", seenAt},
		{prefix + "TEST_PLATE_2", "w", seenAt},
		{prefix + "TEST_PLATE_3", "e", seenAt},
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id := insertRow(t, st, row.plate, row.note, row.seenAt)
		ids = append(ids, id)
	}

	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = st.db.Exec(`DELETE FROM track_events WHERE id = $1`, id)
		}
	})

	// Case no pagination, filtered
	got, err := st.list(0, 0, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}
	if got[0].ID != ids[0] || got[1].ID != ids[1] || got[2].ID != ids[2] {
		t.Fatalf("ids got=%v want=%v", []int64{got[0].ID, got[1].ID, got[2].ID}, ids)
	}

	// Case limit
	got, err = st.list(0, 2, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].ID != ids[0] || got[1].ID != ids[1] {
		t.Fatalf("limit ids got=%v want=%v", []int64{got[0].ID, got[1].ID}, ids[:2])
	}

	// Case offset + limit
	got, err = st.list(1, 2, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].ID != ids[1] || got[1].ID != ids[2] {
		t.Fatalf("offset ids got=%v want=%v", []int64{got[0].ID, got[1].ID}, ids[1:])
	}

	for i := range got {
		if got[i].SeenAt.IsZero() {
			t.Fatalf("SeenAt is zero at i=%d", i)
		}
	}
}

func getSt(t *testing.T) *store {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	st, err := openStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.db.Close() })

	return st
}

func insertRow(t *testing.T, st *store, plate, note string, seenAt time.Time) int64 {
	var id int64

	err := st.db.QueryRow(
		`INSERT INTO track_events (plate, note, seen_at) VALUES ($1, $2, $3) RETURNING id`,
		plate, note, seenAt,
	).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}
