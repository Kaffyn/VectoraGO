package auth

import (
	"testing"
	"time"
)

const testSecretKey = "test-secret-key-for-testing"
const testIssuer = "test-issuer"

func TestNewJWTManager(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	if manager.secretKey != testSecretKey {
		t.Errorf("Expected secretKey '%s', got '%s'", testSecretKey, manager.secretKey)
	}

	if manager.issuer != testIssuer {
		t.Errorf("Expected issuer '%s', got '%s'", testIssuer, manager.issuer)
	}

	if manager.expirationTime != 1*time.Hour {
		t.Errorf("Expected expirationTime 1h, got %v", manager.expirationTime)
	}
}

func TestGenerateToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &Claims{
		UserID:      "user-123",
		Email:       "user@example.com",
		TenantID:    "tenant-456",
		Roles:       []string{"user", "admin"},
		Permissions: []string{"read", "write"},
	}

	token, err := manager.GenerateToken(claims)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Verify token format (three parts separated by dots)
	parts := 0
	for i, c := range token {
		if c == '.' {
			parts++
		}
		if i > 500 { // Reasonable token length check
			break
		}
	}

	if parts != 2 {
		t.Errorf("Expected 2 dots in token, got %d", parts)
	}
}

func TestGenerateTokenWithoutUserID(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &Claims{
		Email: "user@example.com",
	}

	_, err := manager.GenerateToken(claims)

	if err == nil {
		t.Error("Expected error when user_id is missing")
	}
}

func TestGenerateTokenWithNilClaims(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	_, err := manager.GenerateToken(nil)

	if err == nil {
		t.Error("Expected error for nil claims")
	}
}

func TestValidateToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &Claims{
		UserID:      "user-123",
		Email:       "user@example.com",
		TenantID:    "tenant-456",
		Roles:       []string{"user"},
		Permissions: []string{"read"},
	}

	token, _ := manager.GenerateToken(claims)

	validatedClaims, err := manager.ValidateToken(token)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if validatedClaims.UserID != claims.UserID {
		t.Errorf("Expected UserID '%s', got '%s'", claims.UserID, validatedClaims.UserID)
	}

	if validatedClaims.Email != claims.Email {
		t.Errorf("Expected Email '%s', got '%s'", claims.Email, validatedClaims.Email)
	}

	if validatedClaims.TenantID != claims.TenantID {
		t.Errorf("Expected TenantID '%s', got '%s'", claims.TenantID, validatedClaims.TenantID)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	_, err := manager.ValidateToken("invalid-token")

	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &Claims{
		UserID: "user-123",
	}

	token, _ := manager.GenerateToken(claims)

	// Try to validate with wrong secret
	wrongManager := NewJWTManager("wrong-secret-key", testIssuer, 1*time.Hour, 24*time.Hour)
	_, err := wrongManager.ValidateToken(token)

	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	refreshToken, err := manager.GenerateRefreshToken("user-123", "tenant-456")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if refreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}

	// Validate refresh token
	claims, err := manager.ValidateToken(refreshToken)
	if err != nil {
		t.Errorf("Expected valid refresh token, got error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", claims.UserID)
	}
}

func TestRefreshAccessToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	refreshToken, _ := manager.GenerateRefreshToken("user-123", "tenant-456")

	newAccessToken, err := manager.RefreshAccessToken(refreshToken, []string{"admin"}, []string{"write"})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if newAccessToken == "" {
		t.Error("Expected non-empty access token")
	}

	// Validate new access token
	claims, _ := manager.ValidateToken(newAccessToken)
	if !contains(claims.Roles, "admin") {
		t.Error("Expected 'admin' role in new token")
	}
}

func TestClaimsHasRole(t *testing.T) {
	claims := &Claims{
		Roles: []string{"user", "admin"},
	}

	if !claims.HasRole("admin") {
		t.Error("Expected HasRole('admin') to return true")
	}

	if claims.HasRole("superuser") {
		t.Error("Expected HasRole('superuser') to return false")
	}
}

func TestClaimsHasPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"read", "write"},
	}

	if !claims.HasPermission("read") {
		t.Error("Expected HasPermission('read') to return true")
	}

	if claims.HasPermission("delete") {
		t.Error("Expected HasPermission('delete') to return false")
	}
}

func TestClaimsValid(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	claims := &Claims{
		UserID: "user-123",
	}

	manager.GenerateToken(claims)

	// After GenerateToken, claims should be valid
	if err := claims.Valid(); err != nil {
		t.Errorf("Expected valid claims, got error: %v", err)
	}
}

func TestExpiredToken(t *testing.T) {
	// Create manager with very short expiration
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Millisecond, 24*time.Hour)

	claims := &Claims{
		UserID: "user-123",
	}

	token, _ := manager.GenerateToken(claims)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err := manager.ValidateToken(token)

	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestEmptyTokenString(t *testing.T) {
	manager := NewJWTManager(testSecretKey, testIssuer, 1*time.Hour, 24*time.Hour)

	_, err := manager.ValidateToken("")

	if err == nil {
		t.Error("Expected error for empty token string")
	}
}

func contains(arr []string, item string) bool {
	for _, a := range arr {
		if a == item {
			return true
		}
	}
	return false
}
