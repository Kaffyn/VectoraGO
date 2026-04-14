package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kaffyn/Vectora/internal/core/manager"
)

// ChatRequest representa uma requisição de chat
type ChatRequest struct {
	Query        string `json:"query"`
	WorkspaceID  string `json:"workspace_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Temperature  float32 `json:"temperature,omitempty"`
}

// ChatResponse representa a resposta de uma query
type ChatResponse struct {
	Answer         string    `json:"answer"`
	Model          string    `json:"model"`
	TokensUsed     int       `json:"tokens_used"`
	ProcessingTime float64   `json:"processing_time_ms"`
	Sources        []string  `json:"sources,omitempty"`
}

// Validate valida a requisição de chat
func (cr *ChatRequest) Validate() error {
	if cr.Query == "" {
		return fmt.Errorf("query is required")
	}

	if cr.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}

	if cr.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be >= 0")
	}

	if cr.Temperature < 0 || cr.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0 and 2.0")
	}

	return nil
}

// ChatHandler processa requisições de chat
type ChatHandler struct {
	tenantMgr *manager.TenantManager
}

// NewChatHandler cria um novo chat handler
func NewChatHandler(tm *manager.TenantManager) *ChatHandler {
	return &ChatHandler{
		tenantMgr: tm,
	}
}

// HandleChat processa POST /api/chat
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)
	userID := GetUserIDFromContext(r)

	var req ChatRequest
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
		WriteError(w, http.StatusNotFound, fmt.Sprintf("workspace not found or not active: %v", err), requestID)
		return
	}

	// Processa a query usando o engine do tenant
	start := time.Now()
	answer, model, err := tenant.Engine.Query(r.Context(), req.Query, req.WorkspaceID, "", "chat", "standard")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("engine query failed: %v", err), requestID)
		return
	}

	resp := ChatResponse{
		Answer:         answer,
		Model:          model,
		TokensUsed:     0, // TODO: Obter do engine se disponível
		ProcessingTime: float64(time.Since(start).Milliseconds()),
	}

	WriteJSON(w, http.StatusOK, resp, requestID)
}

// HandleStreamChat processa POST /api/chat/stream com respostas em streaming
func (h *ChatHandler) HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err), requestID)
		return
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}

	// Preparar header para streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Request-ID", requestID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "streaming not supported", requestID)
		return
	}

	// TODO: Integrar com engine streaming
	// Mock response em streaming
	messages := []string{
		"The ",
		"actual ",
		"implementation ",
		"will ",
		"stream ",
		"tokens ",
		"here.",
	}

	for _, msg := range messages {
		event := fmt.Sprintf("data: {\"token\":\"%s\",\"request_id\":\"%s\"}\n\n", msg, requestID)
		fmt.Fprint(w, event)
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)
	}
}

// HandleChatHistory retorna o histórico de uma conversa
func (h *ChatHandler) HandleChatHistory(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		WriteError(w, http.StatusBadRequest, "conversation_id is required", requestID)
		return
	}

	// TODO: Recuperar do conversation store
	mockHistory := map[string]interface{}{
		"conversation_id": conversationID,
		"messages": []map[string]interface{}{
			{
				"role":      "user",
				"content":   "What is this?",
				"timestamp": time.Now().Add(-5 * time.Minute).Unix(),
			},
			{
				"role":      "assistant",
				"content":   "This is a mock response.",
				"timestamp": time.Now().Add(-4 * time.Minute).Unix(),
			},
		},
	}

	WriteJSON(w, http.StatusOK, mockHistory, requestID)
}
