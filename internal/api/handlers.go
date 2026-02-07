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
	writePipeline *core.WritePipeline
	readPipeline  *core.ReadPipeline
	maxWriteItems int
}

func NewHandler(writePipeline *core.WritePipeline, readPipeline *core.ReadPipeline, maxWriteItems int) *Handler {
	if writePipeline == nil {
		panic("writePipeline cannot be nil")
	}
	if readPipeline == nil {
		panic("readPipeline cannot be nil")
	}
	if maxWriteItems <= 0 {
		maxWriteItems = 100
	}
	return &Handler{
		writePipeline: writePipeline,
		readPipeline:  readPipeline,
		maxWriteItems: maxWriteItems,
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
	if len(req.Memories) > h.maxWriteItems {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("memories array exceeds maximum of %d items", h.maxWriteItems))
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

	result, err := h.writePipeline.Write(r.Context(), req.TenantID, req.AppID, req.UserID, inputs)
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

func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	var req ReadRequest
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
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	input := core.ReadInput{
		Query: req.Query,
		Types: req.Types,
		Limit: req.Limit,
	}

	result, err := h.readPipeline.Read(r.Context(), req.TenantID, req.AppID, req.UserID, input)
	if err != nil {
		if isReadValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("read pipeline error: tenant=%q app=%q user=%q error=%v", req.TenantID, req.AppID, req.UserID, err)
		writeError(w, http.StatusInternalServerError, sanitizeError(err))
		return
	}

	memories := make([]MemoryOutput, len(result.Memories))
	for i, mem := range result.Memories {
		memories[i] = MemoryOutput{
			ID:              mem.ID.String(),
			Type:            mem.MemoryType,
			Content:         mem.Content,
			ImportanceScore: mem.ImportanceScore,
			Stability:       mem.MemoryStability,
			Metadata:        mem.Metadata,
			CreatedAt:       mem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       mem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp := ReadResponse{
		Memories: memories,
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

func isReadValidationError(err error) bool {
	return errors.Is(err, core.ErrMissingTenantID) ||
		errors.Is(err, core.ErrMissingAppID) ||
		errors.Is(err, core.ErrMissingUserID) ||
		errors.Is(err, core.ErrMissingQuery) ||
		errors.Is(err, core.ErrEmptyQuery) ||
		errors.Is(err, core.ErrQueryTooLong) ||
		errors.Is(err, core.ErrInvalidMemoryType)
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
	if errors.Is(err, core.ErrMemorySearch) {
		return "database error: failed to search memories"
	}
	return "internal server error"
}
