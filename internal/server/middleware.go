package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RequestIDMiddleware adiciona um X-Request-ID único para cada request
func RequestIDMiddleware(headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(headerName)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Adicionar ao context e header
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			w.Header().Set(headerName, requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PanicRecoveryMiddleware recupera de panics e retorna erro 500
func PanicRecoveryMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := r.Context().Value("request_id")
					fmt.Printf("[PANIC] Request %v: %v\n", requestID, err)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error":"internal server error","request_id":"%v"}`, requestID)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLoggingMiddleware registra informações sobre cada request
func RequestLoggingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := r.Context().Value("request_id")

			// Wrap response writer para capturar status code
			wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			fmt.Printf("[%s] %s %s %d %v\n",
				requestID, r.Method, r.RequestURI, wrapped.statusCode, duration)
		})
	}
}

// TimeoutMiddleware adiciona timeout ao context
func TimeoutMiddleware(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BodySizeLimitMiddleware limita o tamanho do request body
func BodySizeLimitMiddleware(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adiciona headers CORS
func CORSMiddleware(allowOrigins []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Verificar se origin está na lista de permitidos
			allowed := false
			for _, ao := range allowOrigins {
				if ao == "*" || ao == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriterWrapper para capturar status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// readBodyWithLimit lê o body com limite de tamanho
func readBodyWithLimit(r *http.Request, maxBytes int64) ([]byte, error) {
	defer r.Body.Close()
	r.Body = io.NopCloser(io.LimitReader(r.Body, maxBytes))
	return io.ReadAll(r.Body)
}
