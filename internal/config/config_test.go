package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("HTTP_PORT", ":9000")
	t.Setenv("DB_HOST", "dbhost")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "dbuser")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "dbname")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://example.com")

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
}
