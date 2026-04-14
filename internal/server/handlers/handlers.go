package handlers

import (
	"encoding/json"
	"net/http"
	"time"
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
	if id := r.Context().Value("request_id"); id != nil {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return ""
}
