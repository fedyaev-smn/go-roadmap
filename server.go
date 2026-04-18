package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func runHTTPServer(addr string, handler http.Handler, onShutdown func(context.Context) error) error {
	if strings.HasPrefix(addr, ":") {
		log.Printf("listening on http://localhost%s", addr)
	} else {
		log.Printf("listening on http://%s", addr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErr <- nil
			return
		}
		serverErr <- err
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown requested")
	case err := <-serverErr:
		if err != nil {
			log.Printf("server error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	if onShutdown != nil {
		_ = onShutdown(shutdownCtx)
	}

	if err := <-serverErr; err != nil {
		return err
	}
	return nil
}
