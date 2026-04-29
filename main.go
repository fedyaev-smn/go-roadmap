package main

import (
	"context"
	"log"
	"net/http"
	"os"
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
	registerRoutes(mux)

	addr := listenAddr()
	if err := runHTTPServer(addr, mux, func(_ context.Context) error {
		return st.db.Close()
	}); err != nil {
		log.Fatal(err)
	}
}
