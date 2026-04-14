package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kaffyn/Vectora/internal/auth"
)

const testSecret = "test-secret-key"
const testIssuer = "test-issuer"

func TestJWTAuthMiddleware(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	// Create claims and generate token
	claims := &auth.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Roles:    []string{"user"},
	}
	token, _ := jwtManager.GenerateToken(claims)

	// Create middleware and handler
	middleware := JWTAuthMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	}))

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "authenticated") {
		t.Errorf("Expected 'authenticated' in response, got: %s", w.Body.String())
	}
}

func TestJWTAuthMiddlewareMissingHeader(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	middleware := JWTAuthMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "missing authorization header") {
		t.Errorf("Expected error message in response")
	}
}

func TestJWTAuthMiddlewareInvalidFormat(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	middleware := JWTAuthMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with invalid format (missing Bearer)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidToken")
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestJWTAuthMiddlewareInvalidToken(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	middleware := JWTAuthMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with invalid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-xyz")
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestOptionalJWTMiddleware(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &auth.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
	}
	token, _ := jwtManager.GenerateToken(claims)

	middleware := OptionalJWTMiddleware(jwtManager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsAuthenticated(r) {
			w.Write([]byte("authenticated"))
		} else {
			w.Write([]byte("not authenticated"))
		}
	}))

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "authenticated") {
		t.Errorf("Expected 'authenticated', got: %s", w.Body.String())
	}

	// Test without token
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Request-ID", "test-123")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if !strings.Contains(w2.Body.String(), "not authenticated") {
		t.Errorf("Expected 'not authenticated', got: %s", w2.Body.String())
	}
}

func TestRoleMiddleware(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &auth.Claims{
		UserID: "user-123",
		Roles:  []string{"admin"},
	}
	token, _ := jwtManager.GenerateToken(claims)

	// Create chain: JWT auth -> Role check
	jwtMiddleware := JWTAuthMiddleware(jwtManager)
	roleMiddleware := RoleMiddleware("admin")

	handler := jwtMiddleware(roleMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authorized"))
	})))

	// Test with required role
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with missing role
	claims2 := &auth.Claims{
		UserID: "user-456",
		Roles:  []string{"user"},
	}
	token2, _ := jwtManager.GenerateToken(claims2)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	req2.Header.Set("X-Request-ID", "test-456")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w2.Code)
	}
}

func TestPermissionMiddleware(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &auth.Claims{
		UserID:      "user-123",
		Permissions: []string{"read", "write"},
	}
	token, _ := jwtManager.GenerateToken(claims)

	jwtMiddleware := JWTAuthMiddleware(jwtManager)
	permMiddleware := PermissionMiddleware("write")

	handler := jwtMiddleware(permMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// Test with required permission
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test without permission
	claims2 := &auth.Claims{
		UserID:      "user-456",
		Permissions: []string{"read"},
	}
	token2, _ := jwtManager.GenerateToken(claims2)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	req2.Header.Set("X-Request-ID", "test-456")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w2.Code)
	}
}

func TestContextExtractors(t *testing.T) {
	jwtManager := auth.NewJWTManager(testSecret, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &auth.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Roles:    []string{"admin"},
	}
	token, _ := jwtManager.GenerateToken(claims)

	jwtMiddleware := JWTAuthMiddleware(jwtManager)
	handler := jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test context extractors
		extractedClaims := GetClaimsFromContext(r)
		extractedUserID := GetUserIDFromContext(r)
		extractedTenantID := GetTenantIDFromContext(r)

		if extractedClaims == nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("no claims"))
			return
		}

		if extractedUserID != "user-123" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("wrong user_id"))
			return
		}

		if extractedTenantID != "tenant-456" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("wrong tenant_id"))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}
