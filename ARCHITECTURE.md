# Vectora Architecture — Phase 2 Restructured

## Overview

Vectora follows a **tiered, categorical architecture** organized into logical layers. Each layer has minimal, controlled dependencies on lower layers.

## Directory Structure

```
internal/
├── config/          ← Tier 1: Configuration & Foundation
│   ├── crypto/      - Workspace crypto/salting
│   ├── i18n/        - Internationalization (translations)
│   ├── policies/    - Policy engine & rule loader
│   └── telemetry/   - Logging & observability
│
├── platform/        ← Tier 2: Platform/OS Layer
│   ├── os/          - OS abstraction (Windows/Linux/Darwin)
│   └── service/     - Singleton pattern, app directory management
│
├── storage/         ← Tier 3: Storage & Database
│   ├── conversations/ - Conversation/message storage
│   ├── db/          - BoltDB KV store, vector storage interface
│   └── infra/       - Infrastructure utilities (config files, logging, sanitizer)
│
├── llm/             ← Tier 4: LLM Providers
│   ├── models.go    - Model registry and metadata
│   ├── provider.go   - Provider interface
│   ├── claude_provider.go  - Anthropic Claude
│   ├── openai_provider.go  - OpenAI GPT
│   ├── gemini_provider.go  - Google Gemini
│   ├── voyage_provider.go  - Voyage AI embeddings
│   ├── gateway.go   - LLM routing & fallback
│   └── context_manager.go - Token counting & context management
│
├── core/            ← Tier 5: Core Logic (Agent & Resource Management)
│   ├── engine/      - Query execution, embedding, RAG orchestration
│   │   ├── engine.go     - Main agent/planning engine
│   │   ├── embedder.go   - Parallel embedding with TurboQuant
│   │   └── turboquant.go - Vector quantization
│   └── manager/     - Tenant/workspace management, resource pools
│       ├── tenant.go      - Tenant context & isolation
│       ├── resource_pool.go - Rate limiting, quota management
│       └── llm_integration.go - LLM binding to tenants
│
├── tools/           ← Tier 5: Tool Execution & Extensions
│   ├── tool.go      - Tool interface & protocol
│   ├── registry.go  - Tool registration & discovery
│   ├── edit.go      - File editing tool
│   ├── read_file.go - File reading tool
│   ├── terminal_run.go - Shell command execution
│   ├── web.go       - Web fetching & browsing
│   └── embedding/   - Tool embedding & RAG
│       ├── embed.go - Vector DB embedding
│       ├── search_database.go - Semantic search
│       ├── bug_pattern_detection.go - Code analysis
│       └── ... (8+ analysis tools)
│
├── api/             ← Tier 6: API & Protocol Layer
│   ├── ipc/         - IPC server (JSON-RPC over stdio/pipes)
│   │   ├── server.go    - IPC server implementation
│   │   ├── router.go    - Route registration & dispatch
│   │   └── transport_*.go - Windows/Unix transport
│   ├── mcp/         - MCP server (Model Context Protocol)
│   │   ├── agent.go     - Agent implementation
│   │   └── stdio.go     - stdio transport
│   ├── acp/         - ACP client (Anthropic Cloud Protocol)
│   │   ├── client.go    - ACP client
│   │   └── server.go    - ACP server bridge
│   ├── jsonrpc/     - JSON-RPC implementation
│   ├── handlers/    - Protocol handlers & middleware
│   └── shared/      - Shared middleware & utilities
│
├── shared/          ← Tier 5+: Shared Utilities
│   ├── git/         - Git repository management
│   ├── ingestion/   - Code parsing & dependency analysis
│   ├── instructions/ - AI instructions & examples
│   └── updater/     - Update checking & management
│
└── tray/            ← Special: CLI/Local Mode UI
    ├── tray.go      - Windows system tray UI
    └── tray_stub.go - Stub for non-Windows platforms
```

## Tier Dependencies

```
Tier 1 (Config)
  └─ Tier 2 (Platform)
     └─ Tier 3 (Storage)
        └─ Tier 4 (LLM)
           └─ Tier 5 (Core + Tools)
              └─ Tier 6 (API)
```

### Tier 1: Configuration (Independent)
- **crypto**: Workspace encryption/salting
- **i18n**: Localization & translations
- **policies**: Policy engine, rule loading, guardians
- **telemetry**: Logging, rotation, observability

**Dependencies**: None (foundation layer)

### Tier 2: Platform (OS Abstraction)
- **os**: Windows/Linux/Darwin abstraction
- **service/singleton**: Singleton management, app directories

**Dependencies**: Tier 1 only

### Tier 3: Storage (Database & Persistence)
- **storage/db**: BoltDB KV store, vector store interfaces
- **storage/infra**: Config files, environment management, logger startup
- **storage/conversations**: Conversation/message storage

**Dependencies**: Tier 1, Tier 2

### Tier 4: LLM (Provider Abstraction)
- Model registry, provider interfaces
- Anthropic Claude, OpenAI, Google Gemini, Voyage AI adapters
- Gateway with routing & fallback logic
- Token counting & context management

**Dependencies**: Tier 1-3

### Tier 5: Core Logic (Agent & Resources)
**Subcomponents:**
- **core/engine**: Query execution, RAG orchestration, embedding
- **core/manager**: Tenant/workspace isolation, resource pools, quota enforcement
- **tools**: Tool registry, execution, 8+ analysis tools (RAG, bug detection, etc.)
- **shared**: Git, code parsing, ingestion, instructions

**Dependencies**: Tier 1-4

### Tier 6: API (Protocol & Communication)
- **api/ipc**: JSON-RPC over IPC (cmd/core interface)
- **api/mcp**: Model Context Protocol (VSCode extension)
- **api/acp**: Anthropic Cloud Protocol (future cloud mode)
- **api/jsonrpc**: Core JSON-RPC implementation

**Dependencies**: Tier 1-5 (all layers)

## Special Modules

### tray (Windows System Tray)
**Location**: `internal/tray`
**Purpose**: Provides system tray UI for CLI/local mode (Windows-specific)
**Dependencies**: core/engine, core/manager, llm, config/i18n
**Status**: Critical for CLI mode — NOT removed, stays separate

## Dependency Flow

```
User Commands (cmd/core)
    ↓
API Layer (internal/api/ipc, mcp)
    ↓
Core Engine (internal/core/engine, manager)
    ↓
Tools (internal/tools)
    ↓
LLM Providers (internal/llm)
    ↓
Storage (internal/storage/db, conversations)
    ↓
Platform (internal/platform/os, singleton)
    ↓
Configuration (internal/config/*)
```

## Key Design Principles

1. **Unidirectional Dependencies**: Lower tiers never depend on higher tiers
2. **Categorical Organization**: Modules grouped by concern (config, platform, storage, etc.)
3. **Clear Boundaries**: Each tier has well-defined responsibilities
4. **Testability**: Each tier can be tested independently
5. **Reusability**: Lower tiers are reusable across different API transports (IPC, MCP, ACP)

## Import Path Structure

All internal imports follow the pattern:
```go
"github.com/Kaffyn/Vectora/internal/{tier}/{module}"
```

Examples:
```go
"github.com/Kaffyn/Vectora/internal/config/crypto"
"github.com/Kaffyn/Vectora/internal/storage/db"
"github.com/Kaffyn/Vectora/internal/core/engine"
"github.com/Kaffyn/Vectora/internal/api/ipc"
```

## Future Extensions

- **HTTP Server**: `internal/server` (Tier 6, for cloud-native mode)
- **Authentication**: `internal/auth` (Tier 6, JWT + RBAC)
- **Observability**: Enhanced telemetry in `internal/config/telemetry`
- **Containerization**: Docker support in `k8s/` and `Dockerfile`

## Build & Verification

All modules verified:
- ✅ No circular imports
- ✅ Unidirectional dependencies
- ✅ All `cmd/` targets compile successfully

```bash
go build ./cmd/core      # CLI mode
go build ./cmd/agent     # Agent mode
go build ./cmd/acp       # ACP client mode
```
