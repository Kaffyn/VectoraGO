package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// ContextKey é o tipo para chaves de context
type ContextKey string

const (
	RequestIDKey    ContextKey = "request_id"
	UserIDKey       ContextKey = "user_id"
	TenantIDKey     ContextKey = "tenant_id"
	AuthenticatedKey ContextKey = "authenticated"
	ClaimsKey       ContextKey = "claims"
)

// Response é a estrutura genérica para respostas da API
type Response struct {
	Status    string        `json:"status"`
	Data      interface{}   `json:"data,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp string        `json:"timestamp"`
	RequestID string        `json:"request_id,omitempty"`
}

// WriteJSON escreve uma resposta JSON com status code
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Status:    http.StatusText(statusCode),
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
	}

	json.NewEncoder(w).Encode(resp)
}

// WriteError escreve uma resposta de erro em JSON
func WriteError(w http.ResponseWriter, statusCode int, errMsg string, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Status:    "error",
		Error:     errMsg,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
	}

	json.NewEncoder(w).Encode(resp)
}

// GetRequestID extrai o Request ID do context
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(RequestIDKey).(string); ok {
		return id
	}
	// Fallback para string pura se middleware não usou a tipagem
	if id, ok := r.Context().Value("request_id").(string); ok {
		return id
	}
	return ""
}

// GetUserIDFromContext extrai o User ID do context
func GetUserIDFromContext(r *http.Request) string {
	if id, ok := r.Context().Value(UserIDKey).(string); ok {
		return id
	}
	if id, ok := r.Context().Value("user_id").(string); ok {
		return id
	}
	return ""
}

// GetTenantIDFromContext extrai o Tenant ID do context
func GetTenantIDFromContext(r *http.Request) string {
	if id, ok := r.Context().Value(TenantIDKey).(string); ok {
		return id
	}
	if id, ok := r.Context().Value("tenant_id").(string); ok {
		return id
	}
	return ""
}
