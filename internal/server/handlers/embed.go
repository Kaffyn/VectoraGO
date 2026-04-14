package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"context"

	"github.com/Kaffyn/Vectora/internal/core/manager"
	"github.com/Kaffyn/Vectora/internal/core/engine"
)

// EmbedRequest representa uma requisição para embeddings
type EmbedRequest struct {
	Paths       []string `json:"paths"`
	WorkspaceID string   `json:"workspace_id"`
	Include     string   `json:"include,omitempty"` // file patterns
	Exclude     string   `json:"exclude,omitempty"` // file patterns
	Force       bool     `json:"force,omitempty"`   // re-embed existing files
}

// EmbedResponse representa a resposta de uma operação de embedding
type EmbedResponse struct {
	JobID          string    `json:"job_id"`
	Status         string    `json:"status"` // queued, processing, completed, failed
	FilesProcessed int       `json:"files_processed"`
	FilesSkipped   int       `json:"files_skipped"`
	TotalChunks    int       `json:"total_chunks"`
	Errors         int       `json:"errors"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	ErrorMsg       string    `json:"error_msg,omitempty"`
}

// EmbedProgress representa o progresso de um embedding job
type EmbedProgress struct {
	JobID           string    `json:"job_id"`
	CurrentFile     string    `json:"current_file"`
	CurrentIndex    int       `json:"current_index"`
	TotalFiles      int       `json:"total_files"`
	Progress        float64   `json:"progress"`
	ElapsedSeconds  float64   `json:"elapsed_seconds"`
	ChunksProcessed int       `json:"chunks_processed"`
	ErrorCount      int       `json:"error_count"`
}

// Validate valida a requisição de embedding
func (er *EmbedRequest) Validate() error {
	if len(er.Paths) == 0 {
		return fmt.Errorf("at least one path is required")
	}

	if er.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}

	return nil
}

// EmbedHandler processa requisições de embedding
type EmbedHandler struct {
	tenantMgr *manager.TenantManager
}

// NewEmbedHandler cria um novo embed handler
func NewEmbedHandler(tm *manager.TenantManager) *EmbedHandler {
	return &EmbedHandler{
		tenantMgr: tm,
	}
}

// HandleStartEmbed processa POST /api/embed/start
func (h *EmbedHandler) HandleStartEmbed(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)
	userID := GetUserIDFromContext(r)

	var req EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err), requestID)
		return
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}

	// Obtém o tenant para o workspace
	tenant, err := h.tenantMgr.GetTenant(req.WorkspaceID)
	if err != nil {
		// Tenta criar se não existir (ou se for o primeiro acesso via API)
		// Para simplicidade, assumimos que o root é o diretório atual se não passado?
		// Na realidade, o workspace deve ser pré-registrado.
		WriteError(w, http.StatusNotFound, fmt.Sprintf("workspace %s not found", req.WorkspaceID), requestID)
		return
	}

	provider := tenant.Engine.LLM.GetDefault()
	if provider == nil || !provider.IsConfigured() {
		WriteError(w, http.StatusPreconditionFailed, "LLM provider not configured for this tenant", requestID)
		return
	}

	// Inicia embedding job assincronamente (semelhante ao IPC)
	jobID := "embed_job_" + requestID[:8]
	
	// Root path default para o root do tenant se não especificado
	rootPath := tenant.Root
	if len(req.Paths) > 0 {
		rootPath = req.Paths[0] // Simplificação: processa o primeiro path
	}

	go engine.RunEmbedJob(
		context.Background(),
		engine.EmbedJobConfig{
			RootPath:       rootPath,
			Include:        req.Include,
			Exclude:        req.Exclude,
			Workspace:      tenant.ID,
			Force:          req.Force,
			CollectionName: "ws_" + tenant.ID, // TODO: Use salter hash if needed
		},
		tenant.KVStore,
		tenant.VectorStore,
		provider,
		func(prog engine.EmbedProgress) {
			// TODO: Store progress in a job store for polling
		},
	)

	resp := EmbedResponse{
		JobID:     jobID,
		Status:    "processing",
		StartedAt: time.Now(),
	}

	WriteJSON(w, http.StatusAccepted, resp, requestID)
}

// HandleEmbedProgress processa GET /api/embed/progress
func (h *EmbedHandler) HandleEmbedProgress(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		WriteError(w, http.StatusBadRequest, "job_id is required", requestID)
		return
	}

	// TODO: Recuperar progresso do job store
	mockProgress := EmbedProgress{
		JobID:          jobID,
		CurrentFile:    "file_150.go",
		CurrentIndex:   150,
		TotalFiles:     500,
		Progress:       30.0,
		ElapsedSeconds: 45.5,
		ChunksProcessed: 2340,
		ErrorCount:     2,
	}

	WriteJSON(w, http.StatusOK, mockProgress, requestID)
}

// HandleEmbedStatus processa GET /api/embed/status
func (h *EmbedHandler) HandleEmbedStatus(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		WriteError(w, http.StatusBadRequest, "job_id is required", requestID)
		return
	}

	// TODO: Recuperar status final do job store
	mockStatus := EmbedResponse{
		JobID:          jobID,
		Status:         "completed",
		FilesProcessed: 500,
		FilesSkipped:   10,
		TotalChunks:    7890,
		Errors:         2,
		StartedAt:      time.Now().Add(-2 * time.Hour),
		CompletedAt:    time.Now(),
	}

	WriteJSON(w, http.StatusOK, mockStatus, requestID)
}

// HandleSearchEmbeddings processa POST /api/embed/search
func (h *EmbedHandler) HandleSearchEmbeddings(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	var searchReq struct {
		Query       string `json:"query"`
		WorkspaceID string `json:"workspace_id"`
		TopK        int    `json:"top_k,omitempty"`
		Threshold   float32 `json:"threshold,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err), requestID)
		return
	}

	if searchReq.Query == "" {
		WriteError(w, http.StatusBadRequest, "query is required", requestID)
		return
	}

	topK := searchReq.TopK
	if topK <= 0 || topK > 100 {
		topK = 10
	}

	// Obtém o tenant para o workspace
	tenant, err := h.tenantMgr.GetTenant(searchReq.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("workspace %s not found", searchReq.WorkspaceID), requestID)
		return
	}

	// 1. Gera embedding da query
	queryVector, err := tenant.Engine.Embed(r.Context(), searchReq.Query, "")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate query embedding: %v", err), requestID)
		return
	}

	// 2. Busca no vector store (coleção ws_ID)
	scoredChunks, err := tenant.VectorStore.Query(r.Context(), "ws_"+tenant.ID, queryVector, topK)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("vector store query failed: %v", err), requestID)
		return
	}

	// 3. Formata resultados
	results := make([]map[string]interface{}, 0, len(scoredChunks))
	for _, sc := range scoredChunks {
		results = append(results, map[string]interface{}{
			"id":        sc.ID,
			"content":   sc.Content,
			"metadata":  sc.Metadata,
			"relevance": sc.Score,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   searchReq.Query,
	}, requestID)
}

// HandleEmbedStats processa GET /api/embed/stats
func (h *EmbedHandler) HandleEmbedStats(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		WriteError(w, http.StatusBadRequest, "workspace_id is required", requestID)
		return
	}

	// TODO: Recuperar estatísticas do vector store
	mockStats := map[string]interface{}{
		"workspace_id":       workspaceID,
		"total_files":        500,
		"total_chunks":       7890,
		"total_embeddings":   7890,
		"storage_mb":         45.2,
		"last_embed_time":    time.Now().Add(-30 * time.Minute),
		"total_embeddings_api_calls": 1250,
		"average_chunk_size": 512,
	}

	WriteJSON(w, http.StatusOK, mockStats, requestID)
}
