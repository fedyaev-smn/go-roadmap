package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

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
