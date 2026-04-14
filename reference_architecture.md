# Vectora — Core Architecture Reference

Este documento serve como referência técnica centralizada para a implementação da Proposta, baseada em `vectora.md`.

## Stack Técnico (Abril 2026)

- **Runtime**: Go 1.23
- **Router**: chi/v5
- **Vector DB**: `usearch-go` v0.2.0 (HNSW)
- **KV Store**: `bbolt` v1.3.11 (BoltDB)
- **Quantização**: TurboQuant (PolarQuant + QJL)
- **Protocolos**: MCP (Model Context Protocol), JSON-RPC 2.0, WebSocket
- **Observabilidade**: OpenTelemetry + zerolog

## Arquitetura de 3 Camadas

### 1. Core Daemon (Local)

Processo leve, orquestrador de agentes, execução de ferramentas em sandbox, suporte a IPC via Unix Sockets/Named Pipes.

- **Entrada**: CLI (cobra/bubbletea) ou MCP stdio.
- **Persistência**: `~/.vectora/db/` (conversations.db e embeddings.db).

### 2. Service (Cloud-Native)

Modo multi-tenant, orquestrado via Kubernetes, com suporte a JWT Auth e RBAC.

- **Endpoints**: `/api/v1/chat`, `/api/v1/plan`, `/api/v1/embed`.
- **Escalabilidade**: Replicado via Deployment, isolamento de usuários no BoltDB.

### 3. Companion (VSCode)

Extensão TypeScript que se conecta ao Core ou Service via WebSockets/MCP.

- **UI**: Webviews para Chat, Memory Browser e Plan Viewer.

## Componentes Internos (`internal/`)

- `core/`: Agent orchestrator, intent classification, planner.
- `llm/`: Abstração de provedores (Anthropic, OpenAI, Gemini).
- `storage/`: Camada KV (Bolt) e Vetorial (usearch).
- `quant/`: Implementação do TurboQuant.
- `tools/`: Registry e executor de ferramentas.
- `mcp/`: Implementação do servidor MCP.
- `auth/`: JWT e RBAC.
- `server/`: Servidor HTTP (Service mode).
- `telemetry/`: OpenTelemetry e logging.

## APIs Públicas (`pkg/`)

- `pkg/vectora/`: SDK oficial do Vectora em Go.
- `pkg/mcp/`: Definições de tipos do protocolo MCP.

---

_Este resumo foi extraído de [vectora.md](file:///c:/Users/bruno/Desktop/Vectora/vectora.md) e serve como base para a refatoração._
