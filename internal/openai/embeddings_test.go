package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eann1s/codex-memory-manager/internal/config"
)

func TestNewEmbeddingProvider(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &config.Config{
			OpenAIAPIKey:         "test-key",
			OpenAIBaseURL:        "https://api.openai.com/v1",
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        30 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
		if provider.apiKey != "test-key" {
			t.Errorf("expected apiKey to be 'test-key', got %s", provider.apiKey)
		}
		if provider.expectedDim != 1536 {
			t.Errorf("expected expectedDim to be 1536, got %d", provider.expectedDim)
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		cfg := &config.Config{
			OpenAIAPIKey:         "",
			OpenAIBaseURL:        "https://api.openai.com/v1",
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        30 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err == nil {
			t.Fatal("expected error for missing api key")
		}
		if provider != nil {
			t.Error("expected provider to be nil on error")
		}
		if !strings.Contains(err.Error(), "OPENAI_API_KEY is required") {
			t.Errorf("expected error message about missing key, got: %v", err)
		}
	})

	t.Run("whitespace only api key", func(t *testing.T) {
		cfg := &config.Config{
			OpenAIAPIKey:         "   ",
			OpenAIBaseURL:        "https://api.openai.com/v1",
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        30 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err == nil {
			t.Fatal("expected error for whitespace api key")
		}
		if provider != nil {
			t.Error("expected provider to be nil on error")
		}
		if !strings.Contains(err.Error(), "cannot be whitespace only") {
			t.Errorf("expected error message about whitespace, got: %v", err)
		}
	})

	t.Run("invalid dimension", func(t *testing.T) {
		cfg := &config.Config{
			OpenAIAPIKey:         "test-key",
			OpenAIBaseURL:        "https://api.openai.com/v1",
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        30 * time.Second,
			EmbeddingDim:         0,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err == nil {
			t.Fatal("expected error for invalid dimension")
		}
		if provider != nil {
			t.Error("expected provider to be nil on error")
		}
		if !strings.Contains(err.Error(), "EMBEDDING_DIM must be greater than 0") {
			t.Errorf("expected error message about dimension, got: %v", err)
		}
	})

	t.Run("base url trailing slash", func(t *testing.T) {
		cfg := &config.Config{
			OpenAIAPIKey:         "test-key",
			OpenAIBaseURL:        "https://api.openai.com/v1/",
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        30 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider.baseURL != "https://api.openai.com/v1" {
			t.Errorf("expected baseURL without trailing slash, got %s", provider.baseURL)
		}
	})
}

func TestEmbed_Success(t *testing.T) {
	expectedEmbedding := make([]float32, 1536)
	for i := range expectedEmbedding {
		expectedEmbedding[i] = float32(i) * 0.001
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("expected path /embeddings, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header with bearer token")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}

		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Embedding: expectedEmbedding,
					Index:     0,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        server.URL,
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	embedding, err := provider.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(embedding) != 1536 {
		t.Errorf("expected embedding length 1536, got %d", len(embedding))
	}

	for i := range embedding {
		if embedding[i] != expectedEmbedding[i] {
			t.Errorf("embedding mismatch at index %d: expected %f, got %f", i, expectedEmbedding[i], embedding[i])
			break
		}
	}
}

func TestEmbed_Validation(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        "https://api.openai.com/v1",
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	t.Run("empty text", func(t *testing.T) {
		_, err := provider.Embed(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty text")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("expected error message about empty text, got: %v", err)
		}
	})

	t.Run("whitespace only text", func(t *testing.T) {
		_, err := provider.Embed(ctx, "   \t\n  ")
		if err == nil {
			t.Fatal("expected error for whitespace text")
		}
		if !strings.Contains(err.Error(), "cannot be whitespace only") {
			t.Errorf("expected error message about whitespace, got: %v", err)
		}
	})
}

func TestEmbed_HTTPErrors(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		shouldRetry    bool
	}{
		{
			name:          "401 unauthorized",
			statusCode:    401,
			responseBody:  `{"error":{"message":"Invalid API key"}}`,
			expectedError: "http 401",
			shouldRetry:   false,
		},
		{
			name:          "429 rate limit",
			statusCode:    429,
			responseBody:  `{"error":{"message":"Rate limit exceeded"}}`,
			expectedError: "http 429",
			shouldRetry:   true,
		},
		{
			name:          "500 internal server error",
			statusCode:    500,
			responseBody:  `{"error":{"message":"Internal error"}}`,
			expectedError: "http 500",
			shouldRetry:   true,
		},
		{
			name:          "503 service unavailable",
			statusCode:    503,
			responseBody:  `{"error":{"message":"Service unavailable"}}`,
			expectedError: "http 503",
			shouldRetry:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attemptCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			cfg := &config.Config{
				OpenAIAPIKey:         "test-key",
				OpenAIBaseURL:        server.URL,
				OpenAIEmbeddingModel: "text-embedding-3-small",
				OpenAITimeout:        5 * time.Second,
				EmbeddingDim:         1536,
			}

			provider, err := NewEmbeddingProvider(cfg)
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}
			provider.backoffFunc = func(attempt int) time.Duration {
				return 0
			}

			ctx := context.Background()
			_, err = provider.Embed(ctx, "test")

			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error to contain '%s', got: %v", tc.expectedError, err)
			}

			if tc.shouldRetry {
				if attemptCount != 3 {
					t.Errorf("expected 3 retry attempts, got %d", attemptCount)
				}
			} else {
				if attemptCount != 1 {
					t.Errorf("expected 1 attempt (no retry), got %d", attemptCount)
				}
			}
		})
	}
}

func TestEmbed_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        server.URL,
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Embed(ctx, "test")

	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "failed to unmarshal response") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestEmbed_DimensionMismatch(t *testing.T) {
	wrongEmbedding := make([]float32, 768)
	for i := range wrongEmbedding {
		wrongEmbedding[i] = 0.1
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Embedding: wrongEmbedding,
					Index:     0,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        server.URL,
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Embed(ctx, "test")

	if err == nil {
		t.Fatal("expected error for dimension mismatch")
	}

	if !strings.Contains(err.Error(), "dimension mismatch") {
		t.Errorf("expected dimension mismatch error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected 1536") {
		t.Errorf("expected error to mention expected dimension 1536, got: %v", err)
	}
	if !strings.Contains(err.Error(), "got 768") {
		t.Errorf("expected error to mention actual dimension 768, got: %v", err)
	}
}

func TestEmbed_RetryBehavior(t *testing.T) {
	t.Run("transient failure then success", func(t *testing.T) {
		attemptCount := 0
		expectedEmbedding := make([]float32, 1536)
		for i := range expectedEmbedding {
			expectedEmbedding[i] = 0.1
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error":{"message":"Service temporarily unavailable"}}`))
				return
			}

			resp := embeddingResponse{
				Data: []struct {
					Embedding []float32 `json:"embedding"`
					Index     int       `json:"index"`
				}{
					{
						Embedding: expectedEmbedding,
						Index:     0,
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := &config.Config{
			OpenAIAPIKey:         "test-key",
			OpenAIBaseURL:        server.URL,
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        5 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}
		provider.backoffFunc = func(attempt int) time.Duration {
			return 0
		}

		ctx := context.Background()
		embedding, err := provider.Embed(ctx, "test")

		if err != nil {
			t.Fatalf("expected success after retry, got error: %v", err)
		}

		if len(embedding) != 1536 {
			t.Errorf("expected embedding length 1536, got %d", len(embedding))
		}

		if attemptCount != 2 {
			t.Errorf("expected 2 attempts (1 failure + 1 success), got %d", attemptCount)
		}
	})

	t.Run("non-retryable failure no retry", func(t *testing.T) {
		attemptCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer server.Close()

		cfg := &config.Config{
			OpenAIAPIKey:         "test-key",
			OpenAIBaseURL:        server.URL,
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAITimeout:        5 * time.Second,
			EmbeddingDim:         1536,
		}

		provider, err := NewEmbeddingProvider(cfg)
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}

		ctx := context.Background()
		_, err = provider.Embed(ctx, "test")

		if err == nil {
			t.Fatal("expected error")
		}

		if attemptCount != 1 {
			t.Errorf("expected 1 attempt (no retry for 401), got %d", attemptCount)
		}
	})
}

func TestEmbed_APIErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddingResponse{
			Error: &apiError{
				Message: "Invalid model specified",
				Type:    "invalid_request_error",
				Code:    "model_not_found",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        server.URL,
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Embed(ctx, "test")

	if err == nil {
		t.Fatal("expected error for API error response")
	}

	if !strings.Contains(err.Error(), "openai api error") {
		t.Errorf("expected API error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Invalid model specified") {
		t.Errorf("expected error message from API, got: %v", err)
	}
}

func TestEmbed_EmptyDataResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAIAPIKey:         "test-key",
		OpenAIBaseURL:        server.URL,
		OpenAIEmbeddingModel: "text-embedding-3-small",
		OpenAITimeout:        5 * time.Second,
		EmbeddingDim:         1536,
	}

	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Embed(ctx, "test")

	if err == nil {
		t.Fatal("expected error for empty data response")
	}

	if !strings.Contains(err.Error(), "no embedding data in response") {
		t.Errorf("expected empty data error, got: %v", err)
	}
}
