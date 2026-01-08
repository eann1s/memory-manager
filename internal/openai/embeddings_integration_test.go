package openai

import (
	"context"
	"os"
	"testing"

	"github.com/eann1s/codex-memory-manager/internal/config"
)

func TestEmbed_Integration(t *testing.T) {
	if os.Getenv("RUN_OPENAI_INTEGRATION") != "1" {
		t.Skip("skipping OpenAI integration test; set RUN_OPENAI_INTEGRATION=1 to run")
	}

	cfg := config.Load()

	if cfg.OpenAIAPIKey == "" {
		t.Fatal("OPENAI_API_KEY must be set for integration test")
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	t.Run("embed hello world", func(t *testing.T) {
		embedding, err := provider.Embed(ctx, "hello world")
		if err != nil {
			t.Fatalf("failed to embed text: %v", err)
		}

		if len(embedding) == 0 {
			t.Fatal("expected non-empty embedding")
		}

		if len(embedding) != cfg.EmbeddingDim {
			t.Errorf("expected embedding dimension %d, got %d", cfg.EmbeddingDim, len(embedding))
		}

		hasNonZero := false
		for _, val := range embedding {
			if val != 0 {
				hasNonZero = true
				break
			}
		}
		if !hasNonZero {
			t.Error("expected at least one non-zero value in embedding")
		}
	})

	t.Run("embed different texts produce different embeddings", func(t *testing.T) {
		embedding1, err := provider.Embed(ctx, "machine learning")
		if err != nil {
			t.Fatalf("failed to embed first text: %v", err)
		}

		embedding2, err := provider.Embed(ctx, "artificial intelligence")
		if err != nil {
			t.Fatalf("failed to embed second text: %v", err)
		}

		if len(embedding1) != len(embedding2) {
			t.Fatalf("embeddings have different lengths: %d vs %d", len(embedding1), len(embedding2))
		}

		identical := true
		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				identical = false
				break
			}
		}

		if identical {
			t.Error("expected different embeddings for different texts")
		}
	})

	t.Run("embed same text produces same embedding", func(t *testing.T) {
		text := "reproducibility test"

		embedding1, err := provider.Embed(ctx, text)
		if err != nil {
			t.Fatalf("failed to embed first time: %v", err)
		}

		embedding2, err := provider.Embed(ctx, text)
		if err != nil {
			t.Fatalf("failed to embed second time: %v", err)
		}

		if len(embedding1) != len(embedding2) {
			t.Fatalf("embeddings have different lengths: %d vs %d", len(embedding1), len(embedding2))
		}

		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				t.Errorf("embeddings differ at index %d: %f vs %f", i, embedding1[i], embedding2[i])
				break
			}
		}
	})
}
