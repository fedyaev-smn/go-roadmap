package main

import (
	"log"
	"net/http"
)

func handleReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	plate := q.Get("plate")

	from, err := parseDateOrRFC3339(q.Get("from"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid from")
		return
	}

	to, err := parseDateOrRFC3339(q.Get("to"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid to")
		return
	}

	items, err := memory.Report(plate, from, to)
	if err != nil {
		log.Printf("report: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	_ = writeJSON(w, http.StatusOK, items)
}
