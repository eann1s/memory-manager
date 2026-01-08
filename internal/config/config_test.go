package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("HTTP_PORT", ":9000")
	t.Setenv("DB_HOST", "dbhost")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "dbuser")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "dbname")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://example.com")
	t.Setenv("OPENAI_EMBEDDING_MODEL", "text-embedding-ada-002")
	t.Setenv("OPENAI_TIMEOUT", "60")
	t.Setenv("EMBEDDING_DIM", "768")

	cfg := Load()

	if cfg.HTTPPort != ":9000" {
		t.Errorf("expected HTTPPort to be ':9000', got %s", cfg.HTTPPort)
	}
	if cfg.DBHost != "dbhost" {
		t.Errorf("expected DBHost to be 'dbhost', got %s", cfg.DBHost)
	}
	if cfg.DBPort != "6543" {
		t.Errorf("expected DBPort to be '6543', got %s", cfg.DBPort)
	}
	if cfg.DBUser != "dbuser" {
		t.Errorf("expected DBUser to be 'dbuser', got %s", cfg.DBUser)
	}
	if cfg.DBPassword != "secret" {
		t.Errorf("expected DBPassword to be 'secret', got %s", cfg.DBPassword)
	}
	if cfg.DBName != "dbname" {
		t.Errorf("expected DBName to be 'dbname', got %s", cfg.DBName)
	}
	if cfg.DBURL != "postgres://dbuser:secret@dbhost:6543/dbname" {
		t.Errorf("expected DBURL to be 'postgres://dbuser:secret@dbhost:6543/dbname', got %s", cfg.DBURL)
	}
	if cfg.OpenAIAPIKey != "test-key" {
		t.Errorf("expected OpenAIAPIKey to be 'test-key', got %s", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://example.com" {
		t.Errorf("expected OpenAIBaseURL to be 'https://example.com', got %s", cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIEmbeddingModel != "text-embedding-ada-002" {
		t.Errorf("expected OpenAIEmbeddingModel to be 'text-embedding-ada-002', got %s", cfg.OpenAIEmbeddingModel)
	}
	if cfg.OpenAITimeout != 60*time.Second {
		t.Errorf("expected OpenAITimeout to be 60s, got %s", cfg.OpenAITimeout)
	}
	if cfg.EmbeddingDim != 768 {
		t.Errorf("expected EmbeddingDim to be 768, got %d", cfg.EmbeddingDim)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	cfg := Load()

	if cfg.HTTPPort != ":8080" {
		t.Errorf("expected default HTTPPort to be ':8080', got %s", cfg.HTTPPort)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected default DBHost to be 'localhost', got %s", cfg.DBHost)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("expected default DBPort to be '5432', got %s", cfg.DBPort)
	}
	if cfg.DBUser != "postgres" {
		t.Errorf("expected default DBUser to be 'postgres', got %s", cfg.DBUser)
	}
	if cfg.DBPassword != "postgres" {
		t.Errorf("expected default DBPassword to be 'postgres', got %s", cfg.DBPassword)
	}
	if cfg.DBName != "postgres" {
		t.Errorf("expected default DBName to be 'postgres', got %s", cfg.DBName)
	}
	if cfg.DBURL != "postgres://postgres:postgres@localhost:5432/postgres" {
		t.Errorf("expected default DBURL to be 'postgres://postgres:postgres@localhost:5432/postgres', got %s", cfg.DBURL)
	}
	if cfg.OpenAIAPIKey != "" {
		t.Errorf("expected default OpenAIAPIKey to be empty, got %s", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default OpenAIBaseURL to be 'https://api.openai.com/v1', got %s", cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIEmbeddingModel != "text-embedding-3-small" {
		t.Errorf("expected default OpenAIEmbeddingModel to be 'text-embedding-3-small', got %s", cfg.OpenAIEmbeddingModel)
	}
	if cfg.OpenAITimeout != 30*time.Second {
		t.Errorf("expected default OpenAITimeout to be 30s, got %s", cfg.OpenAITimeout)
	}
	if cfg.EmbeddingDim != 1536 {
		t.Errorf("expected default EmbeddingDim to be 1536, got %d", cfg.EmbeddingDim)
	}
}
