package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asad110abbas/url-shortener-go/internal/httpapi"
	"github.com/asad110abbas/url-shortener-go/internal/shortener"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := env("ADDR", ":8080")
	store, err := shortener.NewStore(env("DATA_FILE", "links.json"))
	if err != nil {
		slog.Error("load store", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(store, env("BASE_URL", "http://localhost:8080")),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx) // graceful shutdown: finish in-flight requests
	slog.Info("stopped")
}
