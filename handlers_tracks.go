package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

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

func handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseURLID(r.URL, "/tracks/")
	if err != nil {
		if errors.Is(err, errURLPathNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, errURLInvalidID) {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	ev, err := memory.get(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("get track %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	_ = writeJSON(w, http.StatusOK, ev)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseURLID(r.URL, "/tracks/")
	if err != nil {
		if errors.Is(err, errURLPathNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, errURLInvalidID) {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	err = memory.delete(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("delete track %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
