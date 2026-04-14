package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager gerencia criação e validação de tokens JWT
type JWTManager struct {
	secretKey        string
	issuer           string
	expirationTime   time.Duration
	refreshExpiryTime time.Duration
}

// NewJWTManager cria um novo JWT manager
func NewJWTManager(secretKey, issuer string, expiration, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:         secretKey,
		issuer:            issuer,
		expirationTime:    expiration,
		refreshExpiryTime: refreshExpiry,
	}
}

// GenerateToken cria um novo token JWT com os claims fornecidos
func (jm *JWTManager) GenerateToken(claims *Claims) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}

	if claims.UserID == "" {
		return "", errors.New("user_id is required")
	}

	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    jm.issuer,
		Subject:   claims.UserID,
		ExpiresAt: jwt.NewNumericDate(now.Add(jm.expirationTime)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jm.secretKey))
}

// GenerateRefreshToken cria um novo refresh token
func (jm *JWTManager) GenerateRefreshToken(userID, tenantID string) (string, error) {
	if userID == "" {
		return "", errors.New("user_id is required")
	}

	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(jm.refreshExpiryTime)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jm.secretKey))
}

// ValidateToken valida e retorna os claims de um token JWT
func (jm *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token string is empty")
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verificar algoritmo
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jm.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Validar issuer
	if claims.Issuer != jm.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", jm.issuer, claims.Issuer)
	}

	// Validar tempo de expiração
	if err := claims.Valid(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return claims, nil
}

// RefreshAccessToken gera um novo access token a partir de um refresh token válido
func (jm *JWTManager) RefreshAccessToken(refreshToken string, roles, permissions []string) (string, error) {
	refreshClaims, err := jm.ValidateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Criar novos claims com roles e permissions
	newClaims := &Claims{
		UserID:      refreshClaims.UserID,
		Email:       refreshClaims.Email,
		TenantID:    refreshClaims.TenantID,
		Roles:       roles,
		Permissions: permissions,
	}

	return jm.GenerateToken(newClaims)
}

// GetExpirationTime retorna o tempo de expiração configurado
func (jm *JWTManager) GetExpirationTime() time.Duration {
	return jm.expirationTime
}

// GetRefreshExpirationTime retorna o tempo de expiração de refresh token
func (jm *JWTManager) GetRefreshExpirationTime() time.Duration {
	return jm.refreshExpiryTime
}
