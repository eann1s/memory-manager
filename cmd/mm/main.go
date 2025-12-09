package main

import (
	"context"
	"log"
	"net/http"

	"github.com/eann1s/codex-memory-manager/internal/config"
	"github.com/eann1s/codex-memory-manager/internal/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	db, err := store.NewDB(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Pool.Close()

	err = db.Pool.Ping(ctx)
	if err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Default().Printf("Connected to database: %s", cfg.DBURL)

	router := newRouter()

	log.Printf("Memory Manager listening on %s", cfg.HTTPPort)
	if err := http.ListenAndServe(cfg.HTTPPort, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func newRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
