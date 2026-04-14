package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "tracks.db"
	}
	dbPath = filepath.Clean(dbPath)
	st, err := openStore(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	memory = st
	log.Printf("sqlite store: %s", dbPath)

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
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGetByID(w, r)
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
		http.Error(w, `{"error":"invalid limit"}`, http.StatusBadRequest)
		return
	}
	offset, err := parseNonNegQuery(q, "offset")
	if err != nil {
		http.Error(w, `{"error":"invalid offset"}`, http.StatusBadRequest)
		return
	}
	items, err := memory.list(offset, limit, plate)
	if err != nil {
		log.Printf("list tracks: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
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
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	ev, err := memory.get(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("get track %d: %v", id, err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ev)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	items, err := memory.Report()
	if err != nil {
		log.Printf("list tracks: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if body.Plate == "" {
		http.Error(w, `{"error":"plate is required"}`, http.StatusBadRequest)
		return
	}

	ev, err := memory.add(body.Plate, body.Note)
	if err != nil {
		log.Printf("create track: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ev)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health())
}

func handleFixture(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(os.Getenv("FIXTURE")) != "1" {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	items, err := memory.fixture()
	if err != nil {
		log.Printf("fixture: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}
