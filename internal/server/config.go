package server

import "time"

// ServerConfig contém configurações para o HTTP server
type ServerConfig struct {
	Port              int           // Porta HTTP (default: 8080)
	ReadTimeout       time.Duration // Read timeout (default: 30s)
	WriteTimeout      time.Duration // Write timeout (default: 30s)
	ShutdownTimeout   time.Duration // Graceful shutdown timeout (default: 10s)
	MaxHeaderBytes    int           // Max header size (default: 1MB)
	EnableCORS        bool          // Enable CORS (default: false)
	CORSAllowOrigins  []string      // CORS allowed origins
	RequestIDHeader   string        // Header name for request ID (default: X-Request-ID)
	MaxBodySizeBytes  int64         // Max request body size (default: 10MB)
}

// DefaultConfig retorna configuração padrão
func DefaultConfig() ServerConfig {
	return ServerConfig{
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1MB
		EnableCORS:      false,
		RequestIDHeader: "X-Request-ID",
		MaxBodySizeBytes: 10 << 20, // 10MB
	}
}
