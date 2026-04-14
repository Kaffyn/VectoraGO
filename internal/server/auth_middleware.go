package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Kaffyn/Vectora/internal/auth"
	"github.com/Kaffyn/Vectora/internal/server/handlers"
)

// JWTAuthMiddleware valida tokens JWT e extrai claims
func JWTAuthMiddleware(jwtManager *auth.JWTManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extrair token do header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusUnauthorized, "missing authorization header", requestID)
				return
			}

			// Validar formato "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusUnauthorized, "invalid authorization header format", requestID)
				return
			}

			tokenString := parts[1]

			// Validar e parsear token
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err), requestID)
				return
			}

			// Adicionar claims ao context
			ctx := context.WithValue(r.Context(), "claims", claims)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTMiddleware tenta validar JWT mas não bloqueia se falhar
func OptionalJWTMiddleware(jwtManager *auth.JWTManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			ctx := r.Context()
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString := parts[1]
					if claims, err := jwtManager.ValidateToken(tokenString); err == nil {
						ctx = context.WithValue(ctx, "claims", claims)
						ctx = context.WithValue(ctx, "user_id", claims.UserID)
						ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
						ctx = context.WithValue(ctx, "authenticated", true)
					}
				}
			}

			if _, ok := ctx.Value("authenticated").(bool); !ok {
				ctx = context.WithValue(ctx, "authenticated", false)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware verifica se usuário tem um role específico
func RoleMiddleware(requiredRole string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("claims").(*auth.Claims)
			if !ok {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden, "no claims found in context", requestID)
				return
			}

			if !claims.HasRole(requiredRole) {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden,
					fmt.Sprintf("required role '%s' not found", requiredRole), requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PermissionMiddleware verifica se usuário tem uma permissão específica
func PermissionMiddleware(requiredPermission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("claims").(*auth.Claims)
			if !ok {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden, "no claims found in context", requestID)
				return
			}

			if !claims.HasPermission(requiredPermission) {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden,
					fmt.Sprintf("required permission '%s' not found", requiredPermission), requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantMiddleware valida que usuario está acessando seu próprio tenant
func TenantMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("claims").(*auth.Claims)
			if !ok {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden, "no claims found in context", requestID)
				return
			}

			// Pode extrair tenant_id do URL path se necessário
			// Por agora, apenas garante que tenant_id está nos claims
			if claims.TenantID == "" {
				requestID := handlers.GetRequestID(r)
				handlers.WriteError(w, http.StatusForbidden, "no tenant_id in claims", requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaimsFromContext extrai claims do context
func GetClaimsFromContext(r *http.Request) *auth.Claims {
	if claims, ok := r.Context().Value("claims").(*auth.Claims); ok {
		return claims
	}
	return nil
}

// GetUserIDFromContext extrai user_id do context
func GetUserIDFromContext(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// GetTenantIDFromContext extrai tenant_id do context
func GetTenantIDFromContext(r *http.Request) string {
	if tenantID, ok := r.Context().Value("tenant_id").(string); ok {
		return tenantID
	}
	return ""
}

// IsAuthenticated verifica se requisição está autenticada
func IsAuthenticated(r *http.Request) bool {
	if authenticated, ok := r.Context().Value("authenticated").(bool); ok {
		return authenticated
	}
	// Se tem claims, está autenticado
	_, ok := r.Context().Value("claims").(*auth.Claims)
	return ok
}
