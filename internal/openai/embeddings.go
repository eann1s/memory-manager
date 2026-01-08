package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/eann1s/codex-memory-manager/internal/config"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type OpenAIEmbeddingProvider struct {
	apiKey      string
	baseURL     string
	model       string
	expectedDim int
	httpClient  *http.Client
	maxRetries  int
	backoffFunc func(attempt int) time.Duration
}

type embeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func NewEmbeddingProvider(cfg *config.Config) (*OpenAIEmbeddingProvider, error) {
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	if strings.TrimSpace(cfg.OpenAIAPIKey) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY cannot be whitespace only")
	}

	if cfg.EmbeddingDim <= 0 {
		return nil, fmt.Errorf("EMBEDDING_DIM must be greater than 0, got %d", cfg.EmbeddingDim)
	}

	baseURL := strings.TrimRight(cfg.OpenAIBaseURL, "/")

	return &OpenAIEmbeddingProvider{
		apiKey:      cfg.OpenAIAPIKey,
		baseURL:     baseURL,
		model:       cfg.OpenAIEmbeddingModel,
		expectedDim: cfg.EmbeddingDim,
		httpClient: &http.Client{
			Timeout: cfg.OpenAITimeout,
		},
		maxRetries: 3,
		backoffFunc: func(attempt int) time.Duration {
			return time.Duration(attempt+1) * time.Second
		},
	}, nil
}

func (p *OpenAIEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be whitespace only")
	}

	reqBody := embeddingRequest{
		Input: text,
		Model: p.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		embedding, err := p.doRequest(ctx, jsonData)
		if err == nil {
			return embedding, nil
		}

		lastErr = err
		if !p.isRetryable(err) {
			return nil, err
		}

		if attempt < p.maxRetries-1 {
			backoff := p.backoffFunc(attempt)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", p.maxRetries, lastErr)
}

func (p *OpenAIEmbeddingProvider) doRequest(ctx context.Context, jsonData []byte) ([]float32, error) {
	url := p.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(bodyBytes)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       snippet,
		}
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(bodyBytes, &embResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("openai api error: %s (type: %s, code: %s)",
			embResp.Error.Message, embResp.Error.Type, embResp.Error.Code)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	embedding := embResp.Data[0].Embedding
	if len(embedding) != p.expectedDim {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", p.expectedDim, len(embedding))
	}

	return embedding, nil
}

func (p *OpenAIEmbeddingProvider) isRetryable(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == 429 || httpErr.StatusCode >= 500
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Body)
}
