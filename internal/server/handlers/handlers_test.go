package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"message": "test"}
	WriteJSON(w, http.StatusOK, data, "test-request-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got: %s", w.Header().Get("Content-Type"))
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Status != "OK" {
		t.Errorf("Expected status 'OK', got '%s'", resp.Status)
	}

	if resp.RequestID != "test-request-123" {
		t.Errorf("Expected RequestID 'test-request-123', got '%s'", resp.RequestID)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "Invalid input", "test-request-456")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", resp.Status)
	}

	if resp.Error != "Invalid input" {
		t.Errorf("Expected error 'Invalid input', got '%s'", resp.Error)
	}

	if resp.RequestID != "test-request-456" {
		t.Errorf("Expected RequestID 'test-request-456', got '%s'", resp.RequestID)
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "ID present in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "request_id", "test-123")
			},
			expected: "test-123",
		},
		{
			name: "ID not in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "ID in context but wrong type",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "request_id", 123)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req = req.WithContext(tt.setupCtx())

			result := GetRequestID(req)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestResponseStructure(t *testing.T) {
	resp := Response{
		Status:    "success",
		Data:      map[string]string{"key": "value"},
		Error:     "",
		Timestamp: "2026-04-14T10:00:00Z",
		RequestID: "req-123",
	}

	// Verify all fields are serializable
	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal Response: %v", err)
	}

	if len(data) == 0 {
		t.Error("Response marshaled to empty JSON")
	}

	// Verify roundtrip
	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal Response: %v", err)
	}

	if unmarshaled.Status != resp.Status {
		t.Errorf("Status not preserved in roundtrip: %s != %s", unmarshaled.Status, resp.Status)
	}
}

func TestWriteJSONWithNilData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusNoContent, nil, "test-id")

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Data != nil {
		t.Error("Expected Data to be nil for empty response")
	}
}

func TestWriteErrorWithEmpty(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusInternalServerError, "", "test-id")

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Expected empty error, got '%s'", resp.Error)
	}
}
