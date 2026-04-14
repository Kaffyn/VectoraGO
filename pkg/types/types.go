// Package types define os structs comuns compartilhados entre o CLI, core e SDK.
package types

import "time"

// ModelInfo descreve as capacidades de um provedor de LLM.
type ModelInfo struct {
	ID          string   `json:"id"`
	Provider    string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	ContextWindow int     `json:"context_window"`
}

// Stats representa métricas do sistema.
type Stats struct {
	Uptime       time.Duration `json:"uptime"`
	TotalIndexes int           `json:"total_indexes"`
	MemoryUsage  uint64        `json:"memory_usage"`
	AppVersion   string        `json:"app_version"`
}

// WorkspaceMetadata contém informações de auto-descoberta de projetos.
type WorkspaceMetadata struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Language  string    `json:"language"`
	LastIndex time.Time `json:"last_index"`
}
