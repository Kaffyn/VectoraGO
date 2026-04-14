package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// HTTPServer encapsula o servidor HTTP com chi router
type HTTPServer struct {
	config ServerConfig
	router *chi.Mux
	server *http.Server
}

// NewHTTPServer cria um novo HTTPServer com a configuração fornecida
func NewHTTPServer(config ServerConfig) *HTTPServer {
	router := chi.NewRouter()

	// Aplicar middlewares globais
	router.Use(RequestIDMiddleware(config.RequestIDHeader))
	router.Use(PanicRecoveryMiddleware())
	router.Use(RequestLoggingMiddleware())
	router.Use(TimeoutMiddleware(config.ReadTimeout))
	router.Use(BodySizeLimitMiddleware(config.MaxBodySizeBytes))

	if config.EnableCORS {
		router.Use(CORSMiddleware(config.CORSAllowOrigins))
	}

	hs := &HTTPServer{
		config: config,
		router: router,
	}

	// Rotas de health check
	router.Get("/health", hs.handleHealth)
	router.Get("/ready", hs.handleReady)

	return hs
}

// Start inicia o servidor HTTP
func (hs *HTTPServer) Start() error {
	addr := fmt.Sprintf(":%d", hs.config.Port)

	hs.server = &http.Server{
		Addr:           addr,
		Handler:        hs.router,
		ReadTimeout:    hs.config.ReadTimeout,
		WriteTimeout:   hs.config.WriteTimeout,
		MaxHeaderBytes: hs.config.MaxHeaderBytes,
	}

	go func() {
		if err := hs.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	fmt.Printf("HTTP server started on %s\n", addr)
	return nil
}

// Stop para o servidor com graceful shutdown
func (hs *HTTPServer) Stop(ctx context.Context) error {
	if hs.server == nil {
		return nil
	}

	// Usar context timeout se fornecido
	shutdownCtx, cancel := context.WithTimeout(ctx, hs.config.ShutdownTimeout)
	defer cancel()

	return hs.server.Shutdown(shutdownCtx)
}

// GetRouter retorna o chi router para registrar rotas adicionais
func (hs *HTTPServer) GetRouter() *chi.Mux {
	return hs.router
}

// handleHealth retorna status de health do servidor
func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","request_id":"%s"}`,
		time.Now().UTC().Format(time.RFC3339),
		r.Header.Get("X-Request-ID"))
}

// handleReady verifica readiness do servidor (pode ser expandido para verificar dependências)
func (hs *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	// TODO: Verificar dependências (DB, LLM, etc)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"ready":true,"timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}

// RegisterRoute registra uma rota no router (método auxiliar para handlers externos)
func (hs *HTTPServer) RegisterRoute(method, path string, handler http.HandlerFunc) {
	switch method {
	case "GET":
		hs.router.Get(path, handler)
	case "POST":
		hs.router.Post(path, handler)
	case "PUT":
		hs.router.Put(path, handler)
	case "DELETE":
		hs.router.Delete(path, handler)
	case "PATCH":
		hs.router.Patch(path, handler)
	default:
		fmt.Printf("Unsupported HTTP method: %s\n", method)
	}
}

// RegisterMiddleware adiciona um middleware global
func (hs *HTTPServer) RegisterMiddleware(middleware func(next http.Handler) http.Handler) {
	hs.router.Use(middleware)
}
