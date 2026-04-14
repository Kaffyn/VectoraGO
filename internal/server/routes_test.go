package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kaffyn/Vectora/internal/auth"
)

const (
	testJWTSecret = "test-secret-key-for-routes"
	testIssuer    = "test-issuer"
)

func setupTestServer() (*HTTPServer, *auth.JWTManager) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	jwtManager := auth.NewJWTManager(testJWTSecret, testIssuer, 1*time.Hour, 24*time.Hour)
	rbacManager := auth.NewRBACManager()

	server.RegisterRoutes(jwtManager, rbacManager)

	return server, jwtManager
}

func createTestToken(jwtManager *auth.JWTManager, userID, tenantID string, roles []string) string {
	claims := &auth.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
	}
	token, _ := jwtManager.GenerateToken(claims)
	return token
}

func TestChatEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	chatReq := map[string]interface{}{
		"query":       "What is this codebase?",
		"workspace_id": "ws-123",
	}

	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest("POST", "/api/v1/chat/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["answer"] == nil {
		t.Error("Expected 'answer' field in response")
	}
}

func TestChatEndpointUnauthorized(t *testing.T) {
	server, _ := setupTestServer()

	chatReq := map[string]interface{}{
		"query":       "What is this?",
		"workspace_id": "ws-123",
	}

	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest("POST", "/api/v1/chat/", bytes.NewReader(body))
	req.Header.Set("X-Request-ID", "test-456")
	// Sem token

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestStreamChatEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	chatReq := map[string]interface{}{
		"query":       "Stream this",
		"workspace_id": "ws-123",
	}

	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest("POST", "/api/v1/chat/stream", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-stream")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Errorf("Expected Content-Type: text/event-stream, got: %s", w.Header().Get("Content-Type"))
	}
}

func TestPlanEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	planReq := map[string]interface{}{
		"description": "Implement feature X",
		"workspace_id": "ws-123",
		"scope":        "standard",
	}

	body, _ := json.Marshal(planReq)
	req := httptest.NewRequest("POST", "/api/v1/plan/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-plan")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["id"] == nil {
		t.Error("Expected 'id' field in plan response")
	}

	if resp["steps"] == nil {
		t.Error("Expected 'steps' field in plan response")
	}
}

func TestEmbedStartEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	embedReq := map[string]interface{}{
		"paths":        []string{"/src", "/tests"},
		"workspace_id": "ws-123",
		"include":      "*.go,*.md",
	}

	body, _ := json.Marshal(embedReq)
	req := httptest.NewRequest("POST", "/api/v1/embed/start", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-embed")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["job_id"] == nil {
		t.Error("Expected 'job_id' field in embed response")
	}
}

func TestEmbedSearchEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	searchReq := map[string]interface{}{
		"query":        "JWT implementation",
		"workspace_id": "ws-123",
		"top_k":        5,
	}

	body, _ := json.Marshal(searchReq)
	req := httptest.NewRequest("POST", "/api/v1/embed/search", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-search")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["results"] == nil {
		t.Error("Expected 'results' field in search response")
	}
}

func TestAdminEndpoint(t *testing.T) {
	server, jwtManager := setupTestServer()

	// Regular user token (não admin)
	userToken := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("X-Request-ID", "test-admin-1")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for non-admin user, got %d", w.Code)
	}

	// Admin token
	adminToken := createTestToken(jwtManager, "admin-123", "tenant-456", []string{"admin"})

	req2 := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	req2.Header.Set("X-Request-ID", "test-admin-2")

	w2 := httptest.NewRecorder()
	server.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for admin user, got %d", w2.Code)
	}
}

func TestPublicHealthEndpoint(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "healthy" {
		t.Errorf("Expected status='healthy', got '%v'", resp["status"])
	}
}

func TestRequestIDPropagation(t *testing.T) {
	server, jwtManager := setupTestServer()

	token := createTestToken(jwtManager, "user-123", "tenant-456", []string{"user"})

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-propagation-123")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "test-propagation-123" {
		t.Errorf("Expected X-Request-ID to propagate through response")
	}
}
