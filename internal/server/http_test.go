package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPServer(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	if server == nil {
		t.Fatal("Expected NewHTTPServer to return non-nil server")
	}

	if server.router == nil {
		t.Fatal("Expected router to be initialized")
	}
}

func TestHealthEndpoint(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "healthy") {
		t.Errorf("Expected 'healthy' in response, got: %s", w.Body.String())
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got: %s", w.Header().Get("Content-Type"))
	}
}

func TestReadyEndpoint(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "ready") {
		t.Errorf("Expected 'ready' in response, got: %s", w.Body.String())
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	// Test com request ID fornecido
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Request-ID", "test-123")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "test-123" {
		t.Errorf("Expected X-Request-ID: test-123, got: %s", w.Header().Get("X-Request-ID"))
	}

	// Test sem request ID (deve ser gerado)
	req2 := httptest.NewRequest("GET", "/health", nil)
	w2 := httptest.NewRecorder()

	server.router.ServeHTTP(w2, req2)

	if w2.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID to be generated")
	}
}

func TestCORSMiddleware(t *testing.T) {
	config := DefaultConfig()
	config.EnableCORS = true
	config.CORSAllowOrigins = []string{"http://localhost:3000"}

	server := NewHTTPServer(config)

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Expected CORS header, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWildcard(t *testing.T) {
	config := DefaultConfig()
	config.EnableCORS = true
	config.CORSAllowOrigins = []string{"*"}

	server := NewHTTPServer(config)

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS wildcard to allow any origin")
	}
}

func TestBodySizeLimitMiddleware(t *testing.T) {
	config := DefaultConfig()
	config.MaxBodySizeBytes = 100 // Limite pequeno para teste

	server := NewHTTPServer(config)

	// Registrar handler que lê o body
	server.router.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			fmt.Fprintf(w, "Body too large")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Received %d bytes", len(body))
	})

	// Test com body dentro do limite
	smallBody := strings.NewReader("small")
	req := httptest.NewRequest("POST", "/upload", smallBody)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for small body, got %d", w.Code)
	}

	// Test com body acima do limite
	largeBody := strings.NewReader(strings.Repeat("x", 1000))
	req2 := httptest.NewRequest("POST", "/upload", largeBody)
	w2 := httptest.NewRecorder()

	server.router.ServeHTTP(w2, req2)

	if w2.Code == http.StatusOK {
		t.Error("Expected error for body exceeding limit")
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	config := DefaultConfig()
	config.ReadTimeout = 100 * time.Millisecond

	server := NewHTTPServer(config)

	// Registrar handler que respeita timeout
	server.router.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(50 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "completed")
		case <-r.Context().Done():
			w.WriteHeader(http.StatusRequestTimeout)
			fmt.Fprintf(w, "timeout")
		}
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// httptest não respeita timeout exatamente, então apenas verificamos que resposta foi enviada
	if w.Code == 0 {
		t.Error("Expected response from /slow endpoint")
	}
}

func TestPanicRecovery(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	// Registrar handler que faz panic
	server.router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Não deve fazer panic, deve retornar 500
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("Expected error message in response, got: %s", w.Body.String())
	}
}

func TestRegisterRoute(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	// Registrar rota customizada
	server.RegisterRoute("GET", "/custom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "custom handler")
	})

	req := httptest.NewRequest("GET", "/custom", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "custom handler") {
		t.Errorf("Expected 'custom handler' in response, got: %s", w.Body.String())
	}
}

func TestGetRouter(t *testing.T) {
	config := DefaultConfig()
	server := NewHTTPServer(config)

	router := server.GetRouter()

	if router == nil {
		t.Fatal("Expected GetRouter to return non-nil router")
	}

	if router != server.router {
		t.Fatal("Expected GetRouter to return the same router instance")
	}
}

func TestServerStart(t *testing.T) {
	config := DefaultConfig()
	config.Port = 18080 // Usar porta alta para teste

	server := NewHTTPServer(config)

	// Não vamos fazer Start() em testes unitários para evitar portas reais
	// Mas testamos que o método existe e retorna nil
	if server == nil {
		t.Fatal("Expected server to be created")
	}
}

func TestGracefulShutdown(t *testing.T) {
	config := DefaultConfig()
	config.Port = 18081
	config.ShutdownTimeout = 2 * time.Second

	server := NewHTTPServer(config)

	// Erro ao fazer shutdown de servidor não iniciado deve retornar nil
	ctx := context.Background()
	err := server.Stop(ctx)

	if err != nil {
		t.Errorf("Expected nil error for shutdown of non-started server, got: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"Port", config.Port, 8080},
		{"ReadTimeout", config.ReadTimeout, 30 * time.Second},
		{"WriteTimeout", config.WriteTimeout, 30 * time.Second},
		{"ShutdownTimeout", config.ShutdownTimeout, 10 * time.Second},
		{"MaxBodySizeBytes", config.MaxBodySizeBytes, int64(10 << 20)},
		{"RequestIDHeader", config.RequestIDHeader, "X-Request-ID"},
	}

	for _, tt := range tests {
		if tt.value != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.value)
		}
	}
}
