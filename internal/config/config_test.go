package config_test

  import (
      "testing"

      "github.com/eann1s/codex-memory-manager/internal/config"
  )

  func TestLoad(t *testing.T) {
      cases := []struct {
          name        string
          env         map[string]string
          wantPort    string
          wantDBURL   string
          wantBaseURL string
      }{
          {
              name:        "defaults",
              env:         nil,
              wantPort:    ":8080",
              wantDBURL:   "",
              wantBaseURL: "https://api.openai.com/v1",
          },
          {
              name: "overrides",
              env: map[string]string{
                  "HTTP_PORT":      ":9000",
                  "DB_URL":         "postgres://foo",
                  "OPENAI_BASE_URL": "https://custom",
              },
              wantPort:    ":9000",
              wantDBURL:   "postgres://foo",
              wantBaseURL: "https://custom",
          },
      }

      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              for k, v := range tc.env {
                  t.Setenv(k, v)
              }

              cfg := config.Load()
              if cfg.HTTPPort != tc.wantPort {
                  t.Fatalf("HTTPPort = %q, want %q", cfg.HTTPPort, tc.wantPort)
              }
              if cfg.DBURL != tc.wantDBURL {
                  t.Fatalf("DBURL = %q, want %q", cfg.DBURL, tc.wantDBURL)
              }
              if cfg.OpenAIBaseURL != tc.wantBaseURL {
                  t.Fatalf("OpenAIBaseURL = %q, want %q", cfg.OpenAIBaseURL, tc.wantBaseURL)
              }
          })
      }
  }