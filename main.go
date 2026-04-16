package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		log.Fatal("DATABASE_URL is required for Postgres")
	}
	st, err := openStore(dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	memory = st
	log.Printf("postgres store: connected")

	mux := http.NewServeMux()
	// curl.exe -i http://localhost:8080/tracks
	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleList(w, r)
		case http.MethodPost:
			handleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// GET /tracks/{id}
	mux.HandleFunc("/tracks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetByID(w, r)
		case http.MethodDelete:
			handleDelete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleHealth(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleReport(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/fixture", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleFixture(w, r)
	})

	addr := listenAddr()
	if strings.HasPrefix(addr, ":") {
		log.Printf("listening on http://localhost%s", addr)
	} else {
		log.Printf("listening on http://%s", addr)
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		_ = st.db.Close()
		log.Fatal(err)
	}
}

func health() map[string]bool {
	return map[string]bool{"ok": true}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	_ = writeJSON(w, status, map[string]string{"error": msg})
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

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	plate := q.Get("plate")
	limit, err := parseNonNegQuery(q, "limit")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	offset, err := parseNonNegQuery(q, "offset")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid offset")
		return
	}
	items, err := memory.list(offset, limit, plate)
	if err != nil {
		log.Printf("list tracks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}

var errNegativeQueryInt = errors.New("negative")

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
		return 0, errNegativeQueryInt
	}
	return n, nil
}

func handleGetByID(w http.ResponseWriter, r *http.Request) {
	idPart := strings.TrimPrefix(r.URL.Path, "/tracks/")
	idPart = strings.TrimSpace(idPart)
	if idPart == "" || strings.Contains(idPart, "/") {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ev, err := memory.get(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("delete track %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	_ = writeJSON(w, http.StatusOK, ev)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	idPart := strings.TrimPrefix(r.URL.Path, "/tracks/")
	idPart = strings.TrimSpace(idPart)
	if idPart == "" || strings.Contains(idPart, "/") {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = memory.delete(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("get track %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	items, err := memory.Report()
	if err != nil {
		log.Printf("list tracks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Plate == "" {
		writeJSONError(w, http.StatusBadRequest, "plate is required")
		return
	}

	ev, err := memory.add(body.Plate, body.Note)
	if err != nil {
		log.Printf("create track: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	_ = writeJSON(w, http.StatusCreated, ev)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = writeJSON(w, http.StatusOK, health())
}

func handleFixture(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKey(w, r) {
		return
	}

	if strings.TrimSpace(os.Getenv("FIXTURE")) != "1" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	items, err := memory.fixture()
	if err != nil {
		log.Printf("fixture: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
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
