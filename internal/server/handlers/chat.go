package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
	// TODO: Injetar dependências (Engine, Logger, etc)
}

// NewChatHandler cria um novo chat handler
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
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

	// TODO: Validar que usuário tem acesso ao workspace
	// TODO: Chamar engine.Query() para processar
	// TODO: Registrar em conversation store

	// Resposta mockada para testes
	start := time.Now()

	resp := ChatResponse{
		Answer:         "This is a mock response. The actual engine integration will be in Phase 5.",
		Model:          "claude-3.5-sonnet",
		TokensUsed:     150,
		ProcessingTime: time.Since(start).Milliseconds(),
		Sources:        []string{"file1.go", "file2.go"},
	}

	WriteJSON(w, http.StatusOK, resp, requestID)

	// Log para debug
	fmt.Printf("[%s] ChatHandler: user=%s workspace=%s query_len=%d\n",
		requestID, userID, req.WorkspaceID, len(req.Query))
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
