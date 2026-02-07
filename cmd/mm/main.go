package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/eann1s/codex-memory-manager/internal/api"
	"github.com/eann1s/codex-memory-manager/internal/config"
	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/eann1s/codex-memory-manager/internal/openai"
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
	log.Printf("Connected to database")

	repos := store.NewRepos(db.Pool)

	embeddingProvider, err := openai.NewEmbeddingProvider(cfg)
	if err != nil {
		log.Fatalf("failed to initialize embedding provider: %v", err)
	}

	writePipeline := core.NewWritePipeline(repos, embeddingProvider, core.WritePipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxContentLen:       cfg.MemoryMaxContentLen,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	readPipeline := core.NewReadPipeline(repos, embeddingProvider, core.ReadPipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxQueryLen:         cfg.ReadMaxQueryLen,
		MaxItems:            cfg.ReadMaxItems,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	handler := api.NewHandler(writePipeline, readPipeline, cfg.WriteMaxItems)

	router := newRouter(handler)

	server := &http.Server{
		Addr:              cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("Memory Manager listening on %s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func newRouter(handler *api.Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	if handler != nil {
		r.Post("/v1/write", handler.Write)
		r.Post("/v1/read", handler.Read)
	}

	return r
}
