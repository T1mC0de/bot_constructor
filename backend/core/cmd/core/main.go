// Core is the service for sellers (the future website): the HTTP API and the
// storage of the survey and the orders. The bot works through the same API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/api"
	"github.com/T1mC0de/bot_constructor/backend/core/internal/storage/jsonfile"
)

func main() {
	if err := run(); err != nil {
		slog.Error("core stopped with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := getenv("HTTP_ADDR", "localhost:8080")
	surveyFile := getenv("SURVEY_FILE", "data/survey.json")
	ordersFile := getenv("ORDERS_FILE", "data/orders.json")

	for _, file := range []string{surveyFile, ordersFile} {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return err
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(jsonfile.New(surveyFile, ordersFile)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// On Ctrl+C or SIGTERM stop accepting requests and let the current ones finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()

	slog.Info("core started", "addr", addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-shutdownDone
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
