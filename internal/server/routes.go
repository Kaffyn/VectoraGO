package server

import (
	"net/http"

	"github.com/Kaffyn/Vectora/internal/auth"
	"github.com/Kaffyn/Vectora/internal/server/handlers"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registra todas as rotas REST da API
func (hs *HTTPServer) RegisterRoutes(jwtManager *auth.JWTManager, rbacManager *auth.RBACManager) {
	// Handlers
	chatHandler := handlers.NewChatHandler()
	planHandler := handlers.NewPlanHandler()
	embedHandler := handlers.NewEmbedHandler()

	// API v1 rotas
	hs.router.Route("/api/v1", func(r chi.Router) {
		// Public endpoints (sem autenticação)
		r.Get("/health", hs.handleHealth)
		r.Get("/ready", hs.handleReady)

		// Autenticação requerida para tudo abaixo
		r.Use(JWTAuthMiddleware(jwtManager))
		r.Use(TenantMiddleware())

		// Chat endpoints
		r.Route("/chat", func(r chi.Router) {
			r.Post("/", chatHandler.HandleChat)
			r.Post("/stream", chatHandler.HandleStreamChat)
			r.Get("/history", chatHandler.HandleChatHistory)
		})

		// Plan endpoints
		r.Route("/plan", func(r chi.Router) {
			r.Post("/", planHandler.HandleCreatePlan)
			r.Get("/", planHandler.HandleGetPlan)
			r.Patch("/step", planHandler.HandleUpdatePlanStep)
			r.Get("/list", planHandler.HandleListPlans)
		})

		// Embed endpoints
		r.Route("/embed", func(r chi.Router) {
			r.Post("/start", embedHandler.HandleStartEmbed)
			r.Get("/progress", embedHandler.HandleEmbedProgress)
			r.Get("/status", embedHandler.HandleEmbedStatus)
			r.Post("/search", embedHandler.HandleSearchEmbeddings)
			r.Get("/stats", embedHandler.HandleEmbedStats)
		})

		// Admin endpoints (requer admin role)
		r.Route("/admin", func(r chi.Router) {
			r.Use(RoleMiddleware("admin"))

			r.Get("/users", handleAdminUsers)
			r.Get("/workspaces", handleAdminWorkspaces)
			r.Get("/logs", handleAdminLogs)
		})
	})

	// Legacy/backward compatibility rotas (sem /api/v1 prefix)
	hs.router.Route("/chat", func(r chi.Router) {
		r.Use(JWTAuthMiddleware(jwtManager))
		r.Post("/", chatHandler.HandleChat)
	})
}

// handleAdminUsers é um handler stub para uso administrativo
func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	requestID := handlers.GetRequestID(r)
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "List users endpoint - TODO",
		"total":   0,
	}, requestID)
}

// handleAdminWorkspaces é um handler stub para uso administrativo
func handleAdminWorkspaces(w http.ResponseWriter, r *http.Request) {
	requestID := handlers.GetRequestID(r)
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "List workspaces endpoint - TODO",
		"total":   0,
	}, requestID)
}

// handleAdminLogs é um handler stub para uso administrativo
func handleAdminLogs(w http.ResponseWriter, r *http.Request) {
	requestID := handlers.GetRequestID(r)
	handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "List logs endpoint - TODO",
		"total":   0,
	}, requestID)
}

// GetRouterForTesting retorna o chi router (para testes)
func (hs *HTTPServer) GetRouterForTesting() *chi.Mux {
	return hs.router
}
