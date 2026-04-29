package main

import "net/http"

func handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = writeJSON(w, http.StatusOK, health())
}
