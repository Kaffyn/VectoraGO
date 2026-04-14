package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims contém os claims customizados do JWT
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	TenantID    string   `json:"tenant_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// StandardClaims retorna os claims padrão JWT (iss, sub, aud, exp, iat, nbf)
func (c *Claims) StandardClaims() jwt.RegisteredClaims {
	return c.RegisteredClaims
}

// HasRole verifica se claims contém um role específico
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission verifica se claims contém uma permissão específica
func (c *Claims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Valid valida os claims padrão JWT (implementa jwt.Claims interface)
func (c *Claims) Valid() error {
	now := time.Now()

	// Verificar se token não expirou
	if c.ExpiresAt != nil && now.After(c.ExpiresAt.Time) {
		return jwt.ErrTokenExpired
	}

	// Verificar notBefore
	if c.NotBefore != nil && now.Before(c.NotBefore.Time) {
		return jwt.ErrTokenNotValidYet
	}

	return nil
}
