package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eann1s/codex-memory-manager/internal/core"
)

type Handler struct {
	pipeline *core.WritePipeline
	maxItems int
}

func NewHandler(pipeline *core.WritePipeline, maxItems int) *Handler {
	if pipeline == nil {
		panic("pipeline cannot be nil")
	}
	if maxItems <= 0 {
		maxItems = 100
	}
	return &Handler{
		pipeline: pipeline,
		maxItems: maxItems,
	}
}

func (h *Handler) Write(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	var req WriteRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large (max 10MB)")
			return
		}
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain only a single JSON object")
		return
	}

	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.AppID == "" {
		writeError(w, http.StatusBadRequest, "app_id is required")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if len(req.Memories) == 0 {
		writeError(w, http.StatusBadRequest, "memories array cannot be empty")
		return
	}
	if len(req.Memories) > h.maxItems {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("memories array exceeds maximum of %d items", h.maxItems))
		return
	}

	inputs := make([]core.WriteInput, len(req.Memories))
	for i, mem := range req.Memories {
		inputs[i] = core.WriteInput{
			Type:            mem.Type,
			Content:         mem.Content,
			ImportanceScore: mem.ImportanceScore,
			Stability:       mem.Stability,
			Metadata:        mem.Metadata,
		}
	}

	result, err := h.pipeline.Write(r.Context(), req.TenantID, req.AppID, req.UserID, inputs)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("write pipeline error: tenant=%q app=%q user=%q error=%v", req.TenantID, req.AppID, req.UserID, err)
		writeError(w, http.StatusInternalServerError, sanitizeError(err))
		return
	}

	writtenIDs := make([]string, len(result.WrittenIDs))
	for i, id := range result.WrittenIDs {
		writtenIDs[i] = id.String()
	}

	skipped := make([]SkippedItem, len(result.Skipped))
	for i, skip := range result.Skipped {
		skipped[i] = SkippedItem{
			Index:  skip.Index,
			Reason: skip.Reason,
		}
	}

	resp := WriteResponse{
		WrittenIDs: writtenIDs,
		Skipped:    skipped,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func isValidationError(err error) bool {
	return errors.Is(err, core.ErrMissingTenantID) ||
		errors.Is(err, core.ErrMissingAppID) ||
		errors.Is(err, core.ErrMissingUserID)
}

func sanitizeError(err error) string {
	if errors.Is(err, core.ErrTenantResolution) {
		return "database error: failed to resolve tenant"
	}
	if errors.Is(err, core.ErrAppResolution) {
		return "database error: failed to resolve app"
	}
	if errors.Is(err, core.ErrUserResolution) {
		return "database error: failed to resolve user"
	}
	if errors.Is(err, core.ErrEmbeddingGeneration) {
		return "embedding provider error"
	}
	if errors.Is(err, core.ErrEmbeddingDimension) {
		return "embedding dimension mismatch"
	}
	if errors.Is(err, core.ErrMemoryInsertion) {
		return "database error: failed to insert memory"
	}
	return "internal server error"
}
