package main

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Path segment after prefix is missing, empty, or has extra slashes (e.g. /tracks/ or /tracks/1/extra).
var errURLPathNotFound = errors.New("not found")

// Non-numeric or non-positive id in the path (e.g. /tracks/abc).
var errURLInvalidID = errors.New("invalid id")

func health() map[string]bool {
	return map[string]bool{"ok": true}
}

// listenAddr picks where http.ListenAndServe binds.
// ADDR wins if set (full host:port, e.g. ":8080" or "127.0.0.1:8080").
// Else PORT is used as just the port number (e.g. PORT=3000 -> ":3000").
// Else default ":8080".
func listenAddr() string {
	if a := strings.TrimSpace(os.Getenv("ADDR")); a != "" {
		return a
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		return ":" + p
	}
	return ":8080"
}

// parseNonNegQuery returns 0 if key is missing or empty; otherwise parses a non-negative int.
func parseNonNegQuery(q url.Values, key string) (int, error) {
	if !q.Has(key) {
		return 0, nil
	}
	s := strings.TrimSpace(q.Get(key))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errors.New("negative")
	}
	return n, nil
}

// parseURLID parses a single positive int64 path segment after prefix (e.g. prefix "/tracks/" for path "/tracks/42" -> 42).
func parseURLID(u *url.URL, prefix string) (int64, error) {
	idPart := strings.TrimPrefix(u.Path, prefix)
	idPart = strings.TrimSpace(idPart)
	if idPart == "" || strings.Contains(idPart, "/") {
		return 0, errURLPathNotFound
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0, errURLInvalidID
	}
	return id, nil
}

func parseDateOrRFC3339(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// date only
	return time.Parse("2006-01-02", s)
}

func requireAPIKey(w http.ResponseWriter, r *http.Request) bool {
	want := "Bearer " + strings.TrimSpace(os.Getenv("API_KEY"))
	if want == "" {
		writeJSONError(w, http.StatusInternalServerError, "API_KEY not set")
		return false
	}
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	if got == "" || got != want {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}
