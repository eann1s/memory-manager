package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("HTTP_PORT", ":9000")
	t.Setenv("DB_URL", "postgres://localhost:5432/db")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://example.com")

	cfg := Load()

	if cfg.HTTPPort != ":9000" {
		t.Errorf("expected HTTPPort to be ':9000', got %s", cfg.HTTPPort)
	}
	if cfg.DBURL != "postgres://localhost:5432/db" {
		t.Errorf("expected DBURL to be 'postgres://localhost:5432/db', got %s", cfg.DBURL)
	}
	if cfg.OpenAIAPIKey != "test-key" {
		t.Errorf("expected OpenAIAPIKey to be 'test-key', got %s", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://example.com" {
		t.Errorf("expected OpenAIBaseURL to be 'https://example.com', got %s", cfg.OpenAIBaseURL)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DB_URL", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	cfg := Load()

	if cfg.HTTPPort != ":8080" {
		t.Errorf("expected default HTTPPort to be ':8080', got %s", cfg.HTTPPort)
	}
	if cfg.DBURL != "" {
		t.Errorf("expected default DBURL to be empty, got %s", cfg.DBURL)
	}
	if cfg.OpenAIAPIKey != "" {
		t.Errorf("expected default OpenAIAPIKey to be empty, got %s", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default OpenAIBaseURL to be 'https://api.openai.com/v1', got %s", cfg.OpenAIBaseURL)
	}
}
