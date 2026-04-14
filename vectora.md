# Vectora Proposta D — Go Microkernel Architecture

**Documento Técnico Detalhado** | Status: Implementation Guide | Data: Abril 2026

---

## Índice

1. [Filosofia & Posicionamento](#filosofia)
2. [Stack Técnico Completo](#stack)
3. [Arquitetura de Camadas (Core/Service/Companion)](#arquitetura)
4. [Implementação: Core Daemon](#core)
5. [Implementação: Service (Cloud-Native)](#service)
6. [Implementação: Companion VSCode](#companion)
7. [TurboQuant em Go](#turboquant-go)
8. [CLI & Resposta](#cli)
9. [MCP Protocol](#mcp)
10. [Agent Principal Mode](#agent)
11. [Sistema de Treinamento (Harness)](#harness)
12. [Segurança](#segurança)
13. [Git/GitHub Integration](#git)
14. [Telemetry & Observability](#telemetry)
15. [Multi-OS Support](#multiplatform)
16. [Deployment & DevOps](#deployment)

---

<a name="filosofia"></a>

## 1. Filosofia & Posicionamento

**Por que Go?**

- **Concorrência nativa**: goroutines para multi-user Service (1000+ conversas simultâneas sem overhead)
- **Binários estáticos**: uma única binary sem runtime externo. Deploy = `scp vectora /usr/local/bin/`
- **Cloud-native desde o dia zero**: Docker, Kubernetes, cloud providers preferem Go
- **Eco de infra maturo**: melhores practices em logging, tracing, observability já estabelecidas
- **Type safety forte sem ceremony**: Java/C# complexity + Rust verbosity não existem
- **Cross-compilation trivial**: `GOOS=darwin GOARCH=arm64 go build` e pronto

**Não é Go porque:**

- ~~Performance pura (Rust/C++ ganham)~~ — mas Go é "suficientemente rápido"
- ~~ML nativo~~ — Python/Rust melhor, mas Go pode chamar via subprocess/C bindings
- ~~Footprint mínimo absoluto~~ — Zig/C ganha

**É Go porque:**

- Service escalável para 10k+ usuários simultâneos
- Deploy em qualquer lugar (bare metal, Docker, Kubernetes, serverless via CloudRun)
- Equipes hireable (muito mais Go devs que Rust devs)
- Prototipagem + produção no mesmo código

---

<a name="stack"></a>

## 2. Stack Técnico Completo (Proposta D)

### 2.1 Dependências Principais

```go
// go.mod summary
module github.com/kaffyn/vectora

go 1.23

require (
    // Core runtime & Communication
    github.com/go-chi/chi/v5 v5.0.11               // HTTP router (Service mode)
    github.com/gorilla/websocket v1.5.1             // WebSocket para streaming
    github.com/gorilla/rpc v1.2.1                   // JSON-RPC 2.0 (Core IPC)
    google.golang.org/grpc v1.63.0                  // gRPC para comunicação interna
    github.com/sourcegraph/jsonrpc2 v0.1.0          // High-level JSON-RPC clients

    // Protocols
    github.com/mark3labs/mcp-go v0.2.0              // MCP server/client (VSCode protocol)
    // ACP Protocol: implementado manualmente sobre spec BeeAI

    // KV & Vector Storage
    github.com/etcd-io/bbolt v1.3.11               // BoltDB (Core KV)
    github.com/unum-cloud/usearch-go v0.2.0        // HNSW vector indices (CGo)
    github.com/cockroachdb/pebble v1.1.0            // Alternativa para Service em escala

    // Algebra Linear (TurboQuant)
    gonum.org/v1/gonum v0.14.0                      // TurboQuant Go-native algebra
    github.com/willf/bitset v1.1.11                 // Bit operations (QJL)

    // LLM SDKs (Oficiais 2026)
    github.com/anthropics/anthropic-sdk-go v0.1.0
    github.com/sashabaranov/go-openai v1.24.0       // OpenAI v5.4 support
    google.golang.org/api/generativelanguage v0.9.0 // Gemini 3.1 Pro
    github.com/ollama/ollama v0.1.32                // Local Ollama client

    // CLI & TUI (Charmbracelet Stack)
    github.com/spf13/cobra v1.8.0                  // CLI framework
    github.com/charmbracelet/bubbletea v0.25.0     // TUI workflow (Agent Mode)
    github.com/charmbracelet/glamour v0.10.0       // Markdown rendering
    github.com/charmbracelet/bubbles v0.18.0       // UI components
    github.com/charmbracelet/lipgloss v0.10.0      // Styling
    github.com/alecthomas/chroma v2.13.0           // Code syntax highlighting

    // Tray & UI (Wails Stack)
    github.com/getlantern/systray v1.2.2           // System tray (Cross-platform)
    github.com/wailsapp/wails/v3 v3.0.0            // Go + React native app wrapper

    // Safety & Security
    github.com/go-playground/validator v10.20.0    // Input validation
    github.com/monochromegane/go-gitignore v0.0.0  // .vectoraignore parsing
    github.com/golang-jwt/jwt/v5 v5.2.0            // JWT (Service authentication)
    golang.org/x/time/rate v0.5.0                   // Rate limiting

    // Git & GitHub
    github.com/google/go-github/v60 v60.0.0         // GitHub API
    go-git.io/go-git/v5 v5.12.0                     // Local Git operations
    github.com/sergi/go-diff v1.3.1                 // Diff analysis

    // Telemetry
    go.opentelemetry.io/otel v1.26.0               // OpenTelemetry
    github.com/rs/zerolog v1.32.0                  // Zero-allocation logging
)
```

### 2.2 Estrutura do Projeto Go

```
vectora/
├── cmd/
│   ├── vectora/
│   │   └── main.go                    # entry point
│   └── vectora-cli/
│       └── main.go                    # CLI only build
│
├── internal/
│   ├── core/
│   │   ├── agent.go                   # Agent orchestrator
│   │   ├── intent.go                  # Intent classification
│   │   └── planner.go                 # Plan generation
│   │
│   ├── llm/
│   │   ├── provider.go                # LLM abstraction (interface)
│   │   ├── anthropic.go               # Anthropic SDK wrapper
│   │   ├── openai.go                  # OpenAI SDK wrapper
│   │   └── embedding.go               # Embedding orchestration
│   │
│   ├── storage/
│   │   ├── kv.go                      # BoltDB layer
│   │   ├── vector.go                  # usearch wrapper
│   │   ├── cache.go                   # KV Cache (TurboQuant compression)
│   │   └── migrations.go              # schema versioning
│   │
│   ├── quant/
│   │   ├── turboquant.go              # TurboQuant encoder/decoder
│   │   ├── polar.go                   # PolarQuant stage
│   │   ├── qjl.go                     # QJL correction stage
│   │   └── codebook.go                # Lloyd-Max codebooks
│   │
│   ├── tools/
│   │   ├── registry.go                # Tool registration
│   │   ├── executor.go                # Tool execution + sandbox
│   │   ├── definitions.go             # Built-in tools (read_file, edit, etc)
│   │   └── manifest.go                # Tool manifest parsing
│   │
│   ├── mcp/
│   │   ├── server.go                  # MCP server impl
│   │   ├── transport.go               # stdio/TCP transport
│   │   └── resources.go               # Resource enumeration
│   │
│   ├── auth/
│   │   ├── jwt.go                     # JWT generation/validation
│   │   ├── rbac.go                    # Role-based access control
│   │   └── credentials.go             # Credential management
│   │
│   ├── safety/
│   │   ├── sanitizer.go               # Secret detection
│   │   ├── sandbox.go                 # Tool execution sandbox
│   │   └── ignore.go                  # .vectoraignore/.embedignore
│   │
│   ├── git/
│   │   ├── repo.go                    # go-git wrapper
│   │   ├── github.go                  # GitHub API integration
│   │   └── diff.go                    # Diff analysis
│   │
│   ├── server/
│   │   ├── http.go                    # HTTP server (Service mode)
│   │   ├── middleware.go              # JWT, CORS, rate limit
│   │   ├── handlers.go                # API endpoints
│   │   └── websocket.go               # Streaming responses
│   │
│   ├── config/
│   │   ├── config.go                  # Config struct + loading
│   │   └── defaults.go                # Default values
│   │
│   └── telemetry/
│       ├── metrics.go                 # OpenTelemetry metrics
│       ├── traces.go                  # OTLP exporter
│       └── logs.go                    # zerolog configuration
│
├── pkg/
│   ├── vectora/                       # Exported public API
│   │   ├── client.go                  # Go SDK for Vectora
│   │   └── types.go                   # Common types
│   └── mcp/
│       └── protocol.go                # MCP type definitions
│
├── proto/                             # Protocol Buffer definitions
│   ├── vectora.proto                  # Core messages
│   └── kv_cache.proto                 # KV Cache schema
│
├── testdata/
│   ├── fixtures/                      # Test fixtures
│   └── harness/                       # Harness examples
│
├── scripts/
│   ├── build.sh                       # Cross-compile script
│   ├── install.sh                     # Installation script
│   └── docker-entrypoint.sh           # Container entry
│
├── Dockerfile                         # Multi-stage, final ~20MB
├── docker-compose.yml                 # Local dev setup
├── .golangci.yml                      # Linting config
├── Makefile                           # Build targets
├── go.mod & go.sum                    # Dependency lock
├── README.md                          # User guide
└── ARCHITECTURE.md                    # This file
```

---

<a name="arquitetura"></a>

## 3. Arquitetura de Camadas (Core/Service/Companion)

### 3.1 Core Architecture (Local Daemon)

```
┌─────────────────────────────────────────────────────┐
│  Vectora Core (Single Binary)                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │  CLI Layer (cobra + bubbletea)               │  │
│  │  └─ Subcommands: chat, ask, agent, embed    │  │
│  └──────────────────────────────────────────────┘  │
│                    ↓                                │
│  ┌──────────────────────────────────────────────┐  │
│  │  Core Service Layer                          │  │
│  │  ├─ Agent Orchestrator (planning + exec)     │  │
│  │  ├─ Intent Classification (cosine distance)  │  │
│  │  ├─ LLM Abstraction (provider-agnostic)      │  │
│  │  └─ Tool Registry + Executor                 │  │
│  └──────────────────────────────────────────────┘  │
│           ↓                      ↓                  │
│  ┌──────────────────┐  ┌────────────────────────┐ │
│  │  Storage Layer   │  │  Safety & Validation   │ │
│  │  ├─ BoltDB (KV) │  │  ├─ Sanitizer          │ │
│  │  ├─ usearch (Vec)│  │  ├─ Sandbox           │ │
│  │  └─ TurboQuant  │  │  └─ Ignore Files       │ │
│  └──────────────────┘  └────────────────────────┘ │
│           ↓                      ↓                  │
│  ┌──────────────────────────────────────────────┐  │
│  │  IPC Layer                                   │  │
│  │  ├─ Unix Domain Socket (macOS/Linux)        │  │
│  │  ├─ Named Pipes (Windows)                    │  │
│  │  └─ JSON-RPC 2.0 framing                     │  │
│  └──────────────────────────────────────────────┘  │
│                    ↓                                │
│  ~/.vectora/db/                                     │
│  ├─ conversations.db (BoltDB)                       │
│  ├─ embeddings.db (usearch index)                   │
│  ├─ config.yaml                                     │
│  └─ audit.log (append-only)                         │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 3.2 Service Architecture (Multi-tenant Cloud)

```
┌──────────────────────────────────────────────────────┐
│  Vectora Service (Containerized)                     │
├──────────────────────────────────────────────────────┤
│                                                      │
│  ┌─────────────────────────────────────────────┐   │
│  │  HTTP/REST API Layer (chi router)           │   │
│  │  ├─ POST /api/v1/chat                       │   │
│  │  ├─ POST /api/v1/plan                       │   │
│  │  ├─ POST /api/v1/embed                      │   │
│  │  ├─ GET  /api/v1/memory                     │   │
│  │  └─ WebSocket /ws/agent (streaming)         │   │
│  └─────────────────────────────────────────────┘   │
│               ↓                                      │
│  ┌─────────────────────────────────────────────┐   │
│  │  Middleware Stack                           │   │
│  │  ├─ JWT Auth (extract user_id, roles)      │   │
│  │  ├─ RBAC Enforcement                       │   │
│  │  ├─ CORS (configurable origins)            │   │
│  │  ├─ Rate Limiting (golang.org/x/time/rate)│   │
│  │  └─ Request Logging + Tracing              │   │
│  └─────────────────────────────────────────────┘   │
│               ↓                                      │
│  ┌─────────────────────────────────────────────┐   │
│  │  Multi-tenant KV Layer                      │   │
│  │  ├─ BoltDB com namespacing /user/{id}/...  │   │
│  │  ├─ Isolation em nível de chave             │   │
│  │  ├─ Per-user encryption (opcional)          │   │
│  │  └─ Quota enforcement                       │   │
│  └─────────────────────────────────────────────┘   │
│               ↓                                      │
│  ┌──────────────────┐  ┌──────────────────────┐   │
│  │  Job Queue (BG) │  │  Cache Invalidation │   │
│  │  ├─ Embedding    │  │  (TTL + eviction)    │   │
│  │  ├─ Indexing     │  │                       │   │
│  │  └─ Cleanup      │  │                       │   │
│  └──────────────────┘  └──────────────────────┘   │
│                                                      │
│  Persistence:                                       │
│  ├─ BoltDB (local, mas pode ser) → PostgreSQL     │
│  ├─ usearch indices (per-user)                     │
│  └─ TurboQuant compressed vectors                  │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### 3.3 Companion (VSCode Extension) Architecture

```
┌──────────────────────────────────────────┐
│  VSCode Extension (TypeScript)           │
├──────────────────────────────────────────┤
│                                          │
│  vscode.ExtensionContext                │
│  │                                      │
│  ├─ Command Palette Handlers            │
│  │  ├─ vectora.chat                     │
│  │  ├─ vectora.ask                      │
│  │  ├─ vectora.refactor                 │
│  │  └─ vectora.test                     │
│  │                                      │
│  ├─ Webview Panels                      │
│  │  ├─ Chat UI (React/markdown)         │
│  │  ├─ Memory Browser (tree view)       │
│  │  └─ Plan Viewer (step visualization) │
│  │                                      │
│  ├─ Language Server                     │
│  │  └─ Communicates with Go daemon      │
│  │                                      │
│  └─ Status Bar Items                    │
│     ├─ "Vectora: Ready" / "Waiting..."  │
│     └─ Model selector                   │
│                                          │
│  Transport to Core/Service:              │
│  ├─ MCP stdio (padrão)                  │
│  ├─ WebSocket (para Service remoto)     │
│  └─ HTTP REST (fallback)                │
│                                          │
└──────────────────────────────────────────┘
```

---

<a name="core"></a>

## 4. Implementação: Core Daemon

### 4.1 Entry Point & Initialization

```go
// cmd/vectora/main.go

package main

import (
  "context"
  "fmt"
  "os"

  "github.com/kaffyn/vectora/internal/config"
  "github.com/kaffyn/vectora/internal/core"
  "github.com/kaffyn/vectora/internal/server"
  "github.com/kaffyn/vectora/internal/telemetry"
  "github.com/spf13/cobra"
)

func main() {
  // Load config from ~/.vectora/config.yaml + env overrides
  cfg, err := config.Load()
  if err != nil {
    fmt.Fprintf(os.Stderr, "config error: %v\n", err)
    os.Exit(1)
  }

  // Initialize telemetry (logging + tracing)
  telemetry.Init(cfg)
  defer telemetry.Flush()

  // Root command
  rootCmd := &cobra.Command{
    Use:   "vectora",
    Short: "Vectora: Local-first AI knowledge platform",
  }

  // Subcommands
  rootCmd.AddCommand(
    chatCmd(cfg),
    askCmd(cfg),
    agentCmd(cfg),
    embedCmd(cfg),
    memoryCmd(cfg),
    toolsCmd(cfg),
    harnesCmd(cfg),
    serveCmd(cfg),
    debugCmd(cfg),
  )

  // Execute
  if err := rootCmd.Execute(); err != nil {
    os.Exit(1)
  }
}

func serveCmd(cfg *config.Config) *cobra.Command {
  return &cobra.Command{
    Use:   "serve",
    Short: "Start Vectora Service (HTTP server, multi-tenant)",
    RunE: func(cmd *cobra.Command, args []string) error {
      ctx := context.Background()

      // Initialize storage
      kv, err := storage.NewBoltDB(cfg.DataDir)
      if err != nil {
        return fmt.Errorf("kv init: %w", err)
      }
      defer kv.Close()

      vec, err := storage.NewUsearchIndex(cfg.DataDir)
      if err != nil {
        return fmt.Errorf("vec init: %w", err)
      }
      defer vec.Close()

      // Initialize Core service
      corev, err := core.NewVectora(cfg, kv, vec)
      if err != nil {
        return fmt.Errorf("core init: %w", err)
      }

      // Start HTTP server
      srv := server.NewServer(cfg, corev)
      return srv.ListenAndServe(ctx)
    },
  }
}

func chatCmd(cfg *config.Config) *cobra.Command {
  return &cobra.Command{
    Use:   "chat",
    Short: "Interactive chat with Vectora",
    RunE: func(cmd *cobra.Command, args []string) error {
      // ... chat implementation (vide abaixo)
    },
  }
}

// ... outros comandos
```

### 4.2 Core Service (Agent Orchestrator)

```go
// internal/core/agent.go

package core

import (
  "context"
  "fmt"
  "sort"

  "github.com/kaffyn/vectora/internal/llm"
  "github.com/kaffyn/vectora/internal/storage"
  "github.com/kaffyn/vectora/internal/tools"
)

type Vectora struct {
  config *config.Config
  kv     storage.KVStore
  vec    storage.VectorStore
  llmMgr llm.Manager
  tools  *tools.Registry
  intent *IntentClassifier
}

func NewVectora(cfg *config.Config, kv storage.KVStore, vec storage.VectorStore) (*Vectora, error) {
  // Initialize LLM providers
  llmMgr, err := llm.NewManager(cfg)
  if err != nil {
    return nil, fmt.Errorf("llm manager: %w", err)
  }

  // Initialize tool registry
  toolReg := tools.NewRegistry(kv)

  // Load built-in tools
  if err := toolReg.RegisterBuiltins(); err != nil {
    return nil, fmt.Errorf("builtin tools: %w", err)
  }

  // Initialize intent classifier
  intent := NewIntentClassifier(kv)

  return &Vectora{
    config: cfg,
    kv:     kv,
    vec:    vec,
    llmMgr: llmMgr,
    tools:  toolReg,
    intent: intent,
  }, nil
}

// Chat session: user message → maybe use tools → response
func (v *Vectora) Chat(ctx context.Context, userID string, message string) (<-chan string, error) {
  // 1. Classify intent (heurística leve)
  intent, confidence := v.intent.Classify(message)

  if confidence > 0.9 && intent == "greeting" {
    // Resposta direta sem tools
    responseChan := make(chan string, 1)
    responseChan <- "Olá! Como posso ajudar?"
    close(responseChan)
    return responseChan, nil
  }

  // 2. Retrieve RAG context (se necessário)
  var ragContext string
  if confidence < 0.7 || intent == "code_question" {
    results, err := v.vec.Search(ctx, message, 5)
    if err == nil && len(results) > 0 {
      ragContext = formatSearchResults(results)
    }
  }

  // 3. Generate response via LLM
  systemPrompt := fmt.Sprintf(`Você é um assistente de código local.
Usuário tem arquivos e repositórios locais para análise.
RAG Context (se disponível):
%s`, ragContext)

  responseChan, err := v.llmMgr.Stream(ctx, userID, []string{systemPrompt, message})
  return responseChan, err
}

// Agent mode: task → plan → execute tools → response
func (v *Vectora) Agent(ctx context.Context, userID string, task string, plan bool) (<-chan AgentEvent, error) {
  eventChan := make(chan AgentEvent, 10)

  go func() {
    defer close(eventChan)

    // 1. Planning phase
    planEvents := v.plan(ctx, userID, task)
    for evt := range planEvents {
      eventChan <- evt
    }

    // 2. Execution phase (parallel tool invocations)
    execEvents := v.execute(ctx, userID, planEvents)
    for evt := range execEvents {
      eventChan <- evt
    }
  }()

  return eventChan, nil
}

func (v *Vectora) plan(ctx context.Context, userID string, task string) <-chan AgentEvent {
  eventChan := make(chan AgentEvent)

  go func() {
    defer close(eventChan)

    // Ask LLM to generate execution plan
    prompt := fmt.Sprintf(`
Generate a step-by-step plan to accomplish this task:
"%s"

Available tools:
%s

Return JSON:
{
  "steps": [
    {"tool": "tool_name", "args": {...}, "depends_on": []},
    ...
  ]
}`, task, v.tools.Describe())

    // Stream response (for live plan display)
    respChan, _ := v.llmMgr.Stream(ctx, userID, []string{prompt})

    var planJSON string
    for token := range respChan {
      planJSON += token
    }

    var plan ExecutionPlan
    if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
      eventChan <- AgentEvent{
        Type: "error",
        Data: err.Error(),
      }
      return
    }

    eventChan <- AgentEvent{
      Type: "plan_generated",
      Data: plan,
    }
  }()

  return eventChan
}

func (v *Vectora) execute(ctx context.Context, userID string, plan ExecutionPlan) <-chan AgentEvent {
  eventChan := make(chan AgentEvent)

  go func() {
    defer close(eventChan)

    // Dependency graph: paralelizar onde possível
    executed := make(map[string]interface{})

    for _, step := range plan.Steps {
      // Aguardar dependências
      for _, dep := range step.DependsOn {
        for {
          if _, ok := executed[dep]; ok {
            break
          }
          time.Sleep(100 * time.Millisecond)
        }
      }

      // Executar tool
      eventChan <- AgentEvent{
        Type:   "step_start",
        StepID: step.ID,
        Tool:   step.Tool,
      }

      result, err := v.tools.Execute(ctx, step.Tool, step.Args)
      if err != nil {
        eventChan <- AgentEvent{
          Type:   "step_error",
          StepID: step.ID,
          Error:  err,
        }
        continue
      }

      executed[step.ID] = result

      eventChan <- AgentEvent{
        Type:   "step_complete",
        StepID: step.ID,
        Result: result,
      }
    }
  }()

  return eventChan
}

type AgentEvent struct {
  Type   string
  StepID string
  Tool   string
  Data   interface{}
  Result interface{}
  Error  error
}

type ExecutionPlan struct {
  Steps []ExecutionStep `json:"steps"`
}

type ExecutionStep struct {
  ID        string            `json:"id"`
  Tool      string            `json:"tool"`
  Args      map[string]interface{} `json:"args"`
  DependsOn []string          `json:"depends_on"`
}
```

### 4.3 Intent Classification

```go
// internal/core/intent.go

package core

import (
  "github.com/kaffyn/vectora/internal/storage"
)

type IntentClassifier struct {
  kv storage.KVStore
  // Cache de embeddings de intents pré-computados
  intentPatterns map[string][]string
  cache          map[string]string // hash(query) → intent
}

func NewIntentClassifier(kv storage.KVStore) *IntentClassifier {
  return &IntentClassifier{
    kv: kv,
    intentPatterns: map[string][]string{
      "greeting": {"oi", "olá", "oi tudo bem", "ei", "hey", "bom dia"},
      "code_question": {"o que faz", "como funciona", "explique", "entenda", "o código"},
      "refactor": {"refatore", "refactor", "melhore", "otimize", "simplifique"},
      "test": {"teste", "test", "testes", "unit test"},
      "embed": {"index", "embed", "indexar", "embedar"},
    },
    cache: make(map[string]string),
  }
}

func (ic *IntentClassifier) Classify(message string) (intent string, confidence float32) {
  // 1. Check cache
  hash := hashStr(message)
  if cached, ok := ic.cache[hash]; ok {
    return cached, 0.95
  }

  // 2. Simple heuristic: keyword matching
  lowerMsg := strings.ToLower(message)

  for intent, patterns := range ic.intentPatterns {
    for _, pattern := range patterns {
      if strings.Contains(lowerMsg, pattern) {
        ic.cache[hash] = intent
        return intent, 0.85
      }
    }
  }

  // 3. Fallback: cosine similarity com embedding
  // (se quiser pesado: embeddar a mensagem e comparar com intent centers)

  return "general", 0.5
}

func hashStr(s string) string {
  h := fnv.New32a()
  h.Write([]byte(s))
  return fmt.Sprintf("%x", h.Sum32())
}
```

---

<a name="service"></a>

## 5. Implementação: Service (Cloud-Native)

### 5.1 HTTP Server & Routing

```go
// internal/server/http.go

package server

import (
  "context"
  "net/http"

  "github.com/go-chi/chi/v5"
  "github.com/go-chi/chi/v5/middleware"
  "github.com/rs/zerolog/log"
)

type Server struct {
  config *config.Config
  core   *core.Vectora
  router chi.Router
}

func NewServer(cfg *config.Config, core *core.Vectora) *Server {
  router := chi.NewRouter()

  // Middleware
  router.Use(middleware.RequestID)
  router.Use(middleware.RealIP)
  router.Use(middleware.Logger)
  router.Use(middleware.Recoverer)
  router.Use(middleware.Timeout(30 * time.Second))

  // Custom middleware
  router.Use(JWTMiddleware(cfg))
  router.Use(RateLimitMiddleware())

  // Routes
  router.Route("/api/v1", func(r chi.Router) {
    // Public (health check)
    r.Get("/health", health)

    // Protected
    r.Group(func(r chi.Router) {
      r.Use(AuthRequired)

      r.Post("/chat", chatHandler)
      r.Post("/ask", askHandler)
      r.Post("/agent", agentHandler)
      r.Post("/plan", planHandler)
      r.Post("/embed", embedHandler)

      r.Get("/memory", memoryListHandler)
      r.Post("/memory", memorySaveHandler)
      r.Delete("/memory/{id}", memoryDeleteHandler)

      r.Get("/tools", toolsListHandler)
      r.Get("/tools/{name}", toolInfoHandler)

      r.WebSocket("/ws/agent", wsAgentHandler)
    })
  })

  // MCP endpoint
  router.Get("/mcp", mcpHandler)

  return &Server{
    config: cfg,
    core:   core,
    router: router,
  }
}

func (s *Server) ListenAndServe(ctx context.Context) error {
  srv := &http.Server{
    Addr:    s.config.ListenAddr,
    Handler: s.router,
  }

  go func() {
    <-ctx.Done()
    srv.Shutdown(context.Background())
  }()

  log.Info().Str("addr", s.config.ListenAddr).Msg("server starting")
  return srv.ListenAndServe()
}

// Handlers
func chatHandler(w http.ResponseWriter, r *http.Request) {
  userID := r.Context().Value("user_id").(string)

  var req struct {
    Message string `json:"message"`
  }
  json.NewDecoder(r.Body).Decode(&req)

  // Stream response via SSE
  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")

  respChan, _ := s.core.Chat(r.Context(), userID, req.Message)

  for token := range respChan {
    fmt.Fprintf(w, "data: %s\n\n", token)
    w.(http.Flusher).Flush()
  }
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
  userID := r.Context().Value("user_id").(string)

  var req struct {
    Task string `json:"task"`
    Plan bool   `json:"plan"`
  }
  json.NewDecoder(r.Body).Decode(&req)

  eventChan, _ := s.core.Agent(r.Context(), userID, req.Task, req.Plan)

  // Stream events via SSE
  w.Header().Set("Content-Type", "text/event-stream")

  for evt := range eventChan {
    data, _ := json.Marshal(evt)
    fmt.Fprintf(w, "data: %s\n\n", data)
    w.(http.Flusher).Flush()
  }
}

func wsAgentHandler(w http.ResponseWriter, r *http.Request) {
  // WebSocket para agent mode com feedback em tempo real
  // (vide seção WebSocket abaixo)
}
```

### 5.2 Authentication & RBAC

```go
// internal/auth/jwt.go

package auth

import (
  "errors"
  "time"

  "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
  UserID string
  Email  string
  Roles  []string
  jwt.RegisteredClaims
}

func GenerateToken(userID, email string, roles []string, secret string) (string, error) {
  claims := Claims{
    UserID: userID,
    Email:  email,
    Roles:  roles,
    RegisteredClaims: jwt.RegisteredClaims{
      ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
      IssuedAt:  jwt.NewNumericDate(time.Now()),
    },
  }

  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  return token.SignedString([]byte(secret))
}

func VerifyToken(tokenString string, secret string) (*Claims, error) {
  claims := &Claims{}
  token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
    return []byte(secret), nil
  })

  if err != nil || !token.Valid {
    return nil, errors.New("invalid token")
  }

  return claims, nil
}

// RBAC
type Role string

const (
  RoleAdmin  Role = "admin"
  RoleUser   Role = "user"
  RoleViewer Role = "viewer"
)

func (r Role) CanExecuteAgent() bool {
  return r == RoleAdmin || r == RoleUser
}

func (r Role) CanEmbedProject() bool {
  return r == RoleAdmin || r == RoleUser
}

func (r Role) CanReadMemory() bool {
  return r == RoleAdmin || r == RoleUser || r == RoleViewer
}
```

### 5.3 Multi-tenant KV Namespacing

```go
// internal/storage/kv.go

package storage

import (
  "github.com/etcd-io/bbolt"
)

type BoltDB struct {
  db *bbolt.DB
}

func (b *BoltDB) Put(userID, bucket string, key, value []byte) error {
  return b.db.Update(func(tx *bbolt.Tx) error {
    // Namespace by user: /user/{userID}/{bucket}
    nsKey := []byte("user/" + userID + "/" + bucket)
    bkt, err := tx.CreateBucketIfNotExists(nsKey)
    if err != nil {
      return err
    }
    return bkt.Put(key, value)
  })
}

func (b *BoltDB) Get(userID, bucket string, key []byte) ([]byte, error) {
  var value []byte
  err := b.db.View(func(tx *bbolt.Tx) error {
    nsKey := []byte("user/" + userID + "/" + bucket)
    bkt := tx.Bucket(nsKey)
    if bkt == nil {
      return nil
    }
    value = bkt.Get(key)
    return nil
  })
  return value, err
}

// Range query (exemplo)
func (b *BoltDB) RangeByPrefix(userID, bucket string, prefix []byte) ([][]byte, error) {
  var results [][]byte
  err := b.db.View(func(tx *bbolt.Tx) error {
    nsKey := []byte("user/" + userID + "/" + bucket)
    bkt := tx.Bucket(nsKey)
    if bkt == nil {
      return nil
    }
    c := bkt.Cursor()
    for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
      results = append(results, v)
    }
    return nil
  })
  return results, err
}
```

---

<a name="companion"></a>

## 6. Implementação: Companion VSCode Extension

### 6.1 Extension Manifest & Activation

```typescript
// package.json
{
  "name": "vectora-companion",
  "displayName": "Vectora AI Agent",
  "version": "0.1.0",
  "publisher": "kaffyn",
  "engines": {
    "vscode": "^1.90.0"
  },
  "activationEvents": [
    "onStartupFinished",
    "onCommand:vectora.chat",
    "onCommand:vectora.ask"
  ],
  "contributes": {
    "commands": [
      {
        "command": "vectora.chat",
        "title": "Vectora: Chat",
        "category": "Vectora"
      },
      {
        "command": "vectora.ask",
        "title": "Vectora: Ask about Selection",
        "category": "Vectora"
      },
      {
        "command": "vectora.refactor",
        "title": "Vectora: Refactor Code",
        "category": "Vectora"
      }
    ],
    "views": {
      "explorer": [
        {
          "id": "vectoraMemory",
          "name": "Vectora Memory"
        }
      ]
    },
    "configuration": {
      "properties": {
        "vectora.daemonPath": {
          "type": "string",
          "default": "vectora",
          "description": "Path to Vectora binary"
        },
        "vectora.serviceUrl": {
          "type": "string",
          "description": "Remote Service URL (for Service mode)"
        },
        "vectora.provider": {
          "type": "string",
          "enum": ["anthropic", "openai", "google"],
          "default": "anthropic"
        }
      }
    }
  }
}
```

### 6.2 Extension Main Entry

```typescript
// src/extension.ts

import * as vscode from "vscode";
import { VectoraClient } from "./client";
import { ChatPanel } from "./panels/chatPanel";
import { MemoryTreeProvider } from "./views/memoryTree";

let client: VectoraClient;
let chatPanel: ChatPanel;

export async function activate(context: vscode.ExtensionContext) {
  // Initialize client (daemon ou Service remoto)
  const daemonPath = vscode.workspace
    .getConfiguration("vectora")
    .get<string>("daemonPath", "vectora");
  const serviceUrl = vscode.workspace
    .getConfiguration("vectora")
    .get<string>("serviceUrl");

  client = new VectoraClient(daemonPath, serviceUrl);

  // Command: Chat
  context.subscriptions.push(
    vscode.commands.registerCommand("vectora.chat", async () => {
      if (!chatPanel) {
        chatPanel = new ChatPanel(context.extensionUri, client);
      }
      chatPanel.reveal();
    }),
  );

  // Command: Ask about selection
  context.subscriptions.push(
    vscode.commands.registerCommand("vectora.ask", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor) return;

      const selection = editor.document.getText(editor.selection);
      const response = await client.ask(selection);

      vscode.window.showInformationMessage(response);
    }),
  );

  // Memory tree view
  const memoryProvider = new MemoryTreeProvider(client);
  vscode.window.registerTreeDataProvider("vectoraMemory", memoryProvider);

  // Status bar
  const statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100,
  );
  statusBarItem.text = "$(robot) Vectora: Ready";
  statusBarItem.show();

  context.subscriptions.push(statusBarItem);

  // WebSocket listener (para eventos de daemon)
  client.onEvent((evt) => {
    if (evt.type === "daemon_connected") {
      statusBarItem.text = "$(robot) Vectora: Connected";
    } else if (evt.type === "daemon_disconnected") {
      statusBarItem.text = "$(robot) Vectora: Disconnected";
    }
  });
}

export function deactivate() {
  client?.disconnect();
}
```

### 6.3 Chat Panel (Webview)

```typescript
// src/panels/chatPanel.ts

import * as vscode from "vscode";

export class ChatPanel {
  private panel: vscode.WebviewPanel;
  private client: VectoraClient;

  constructor(extensionUri: vscode.Uri, client: VectoraClient) {
    this.client = client;

    this.panel = vscode.window.createWebviewPanel(
      "vectoraChat",
      "Vectora Chat",
      vscode.ViewColumn.Beside,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
      },
    );

    this.panel.webview.html = this.getHTML();

    // Webview message handling
    this.panel.webview.onDidReceiveMessage(async (message) => {
      if (message.command === "sendMessage") {
        const response = await this.client.chat(message.text);
        this.panel.webview.postMessage({
          type: "responseStream",
          content: response,
        });
      }
    });
  }

  reveal() {
    this.panel.reveal(vscode.ViewColumn.Beside);
  }

  private getHTML(): string {
    return `
<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: monospace; padding: 10px; }
    #messages { border: 1px solid #ccc; height: 300px; overflow-y: auto; margin-bottom: 10px; }
    .message { padding: 5px; margin: 5px 0; }
    .user { background: #e3f2fd; }
    .assistant { background: #f5f5f5; }
    #input { width: 100%; padding: 5px; }
  </style>
</head>
<body>
  <div id="messages"></div>
  <input type="text" id="input" placeholder="Type your question...">
  
  <script>
    const vscode = acquireVsCodeApi();
    
    document.getElementById('input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        const text = document.getElementById('input').value;
        vscode.postMessage({ command: 'sendMessage', text });
        document.getElementById('input').value = '';
      }
    });
    
    window.addEventListener('message', (e) => {
      const { type, content } = e.data;
      if (type === 'responseStream') {
        const div = document.createElement('div');
        div.className = 'message assistant';
        div.innerText = content;
        document.getElementById('messages').appendChild(div);
      }
    });
  </script>
</body>
</html>
    `;
  }
}
```

---

<a name="turboquant-go"></a>

## 7. TurboQuant em Go

### 7.1 PolarQuant Stage

```go
// internal/quant/polar.go

package quant

import (
  "math"
  "github.com/willf/bitset"
  "gonum.org/v1/gonum/mat"
)

type PolarQuant struct {
  headDim int
  kBits   uint8
  vBits   uint8
  R       *mat.Dense      // Rotation matrix [headDim x headDim]
  kCodebook []float32     // Lloyd-Max codebook para K
  vCodebook []float32     // Lloyd-Max codebook para V
}

func NewPolarQuant(headDim int, kBits, vBits uint8) (*PolarQuant, error) {
  pq := &PolarQuant{
    headDim:   headDim,
    kBits:     kBits,
    vBits:     vBits,
    R:         generateRotationMatrix(42, headDim),
    kCodebook: lloydMaxCodebook(kBits),
    vCodebook: lloydMaxCodebook(vBits),
  }
  return pq, nil
}

// CompressKV: compress embedding (ou KV head)
func (pq *PolarQuant) CompressKV(k, v []float32) (*CompressedKV, error) {
  if len(k) != pq.headDim || len(v) != pq.headDim {
    return nil, fmt.Errorf("invalid dimensions")
  }

  // Stage 1: Apply rotation (precondicionamento)
  kRotated := make([]float32, pq.headDim)
  vRotated := make([]float32, pq.headDim)

  for i := 0; i < pq.headDim; i++ {
    var kSum, vSum float32
    for j := 0; j < pq.headDim; j++ {
      kSum += pq.R.At(i, j) * k[j]
      vSum += pq.R.At(i, j) * v[j]
    }
    kRotated[i] = kSum
    vRotated[i] = vSum
  }

  // Compute magnitudes (radius)
  kRadius := norm(kRotated)
  vRadius := norm(vRotated)

  // Normalize to magnitude=1
  kNorm := pq.normalize(kRotated, kRadius)
  vNorm := pq.normalize(vRotated, vRadius)

  // Transform to polar (extract angles)
  kAngles := pq.extractAngles(kNorm)
  vAngles := pq.extractAngles(vNorm)

  // Quantize angles using Lloyd-Max codebook
  kIndices := pq.quantize(kAngles, pq.kCodebook)
  vIndices := pq.quantize(vAngles, pq.vCodebook)

  // Pack into bitstream
  kPacked := packBitstream(kIndices, pq.kBits)
  vPacked := packBitstream(vIndices, pq.vBits)

  return &CompressedKV{
    KPacked:  kPacked,
    VPacked:  vPacked,
    KRadius:  kRadius,
    VRadius:  vRadius,
  }, nil
}

func (pq *PolarQuant) normalize(v []float32, radius float32) []float32 {
  if radius < 1e-6 {
    return v
  }
  normalized := make([]float32, len(v))
  for i := range v {
    normalized[i] = v[i] / radius
  }
  return normalized
}

func (pq *PolarQuant) extractAngles(normalized []float32) []float32 {
  angles := make([]float32, pq.headDim/2)
  for i := 0; i < len(angles); i++ {
    angle := math.Atan2(float64(normalized[2*i+1]), float64(normalized[2*i]))
    angles[i] = float32(angle)
  }
  return angles
}

func (pq *PolarQuant) quantize(values []float32, codebook []float32) []uint8 {
  indices := make([]uint8, len(values))
  for i, v := range values {
    bestIdx := uint8(0)
    bestDist := math.Abs(float64(v - codebook[0]))
    for j, cb := range codebook {
      dist := math.Abs(float64(v - cb))
      if dist < bestDist {
        bestDist = dist
        bestIdx = uint8(j)
      }
    }
    indices[i] = bestIdx
  }
  return indices
}

func packBitstream(indices []uint8, bitwidth uint8) []uint8 {
  bitsPerByte := 8 / bitwidth
  packed := []uint8{}

  for i := 0; i < len(indices); i += int(bitsPerByte) {
    var byte uint8
    for j := 0; j < int(bitsPerByte) && i+j < len(indices); j++ {
      byte |= (indices[i+j] & ((1 << bitwidth) - 1)) << (j * bitwidth)
    }
    packed = append(packed, byte)
  }

  return packed
}

func norm(v []float32) float32 {
  var sum float32
  for _, x := range v {
    sum += x * x
  }
  return float32(math.Sqrt(float64(sum)))
}

// Generative rotation matrix via QR decomposition
func generateRotationMatrix(seed int64, dim int) *mat.Dense {
  // Gerar matriz aleatória A, depois fazer QR(A) → Q é ortogonal
  data := make([]float64, dim*dim)
  rng := rand.New(rand.NewSource(seed))
  for i := range data {
    data[i] = rng.NormFloat64()
  }
  A := mat.NewDense(dim, dim, data)

  var qr mat.QR
  qr.Factorize(A)

  Q := mat.NewDense(dim, dim, nil)
  qr.QTo(Q)

  return Q
}

func lloydMaxCodebook(bitwidth uint8) []float32 {
  numBuckets := 1 << bitwidth
  codebook := make([]float32, numBuckets)
  for i := 0; i < numBuckets; i++ {
    // Simplified: uniform bucketing
    // Production: usar Lloyd-Max iterativo
    codebook[i] = (float32(i) + 0.5) / float32(numBuckets)
  }
  return codebook
}

type CompressedKV struct {
  KPacked  []uint8
  VPacked  []uint8
  KRadius  float32
  VRadius  float32
}
```

### 7.2 QJL Correction Stage

```go
// internal/quant/qjl.go

package quant

import (
  "github.com/willf/bitset"
)

type QJLCorrector struct {
  headDim int
  G       *mat.Dense  // Gaussian matrix [d/2 x d]
}

func NewQJLCorrector(headDim int) *QJLCorrector {
  return &QJLCorrector{
    headDim: headDim,
    G:       generateGaussianMatrix(43, headDim),
  }
}

// ApplyCorrection: adiciona 1-bit QJL correction
func (qjl *QJLCorrector) ApplyCorrection(
  kCompressed *CompressedKV,
  kOriginal []float32,
) (*QJLCorrected, error) {

  // 1. Dequantize (reconstruct quantized values)
  kDequant := dequantize(kCompressed.KPacked, pq.kCodebook, kCompressed.KRadius)

  // 2. Compute residual error
  kError := subtract(kDequant, kOriginal)

  // 3. Project residual via Gaussian matrix
  kErrorProjected := matmul(qjl.G, kError)  // result: [d/2]

  // 4. Extract sign bits (1-bit quantization)
  kSignBits := bitset.New(uint(len(kErrorProjected)))
  for i, v := range kErrorProjected {
    if v > 0 {
      kSignBits.Set(uint(i))
    }
  }

  // Analogous for V...
  vSignBits := bitset.New(uint(len(vErrorProjected)))

  return &QJLCorrected{
    KSignBits: kSignBits,
    VSignBits: vSignBits,
  }, nil
}

// Decompress: reconstruct from compressed + QJL signs
func (qjl *QJLCorrector) Decompress(
  compressed *CompressedKV,
  corrected *QJLCorrected,
) ([]float32, []float32, error) {

  // Inverse: reconstruct error_projected from sign bits
  kSignValues := make([]float32, compressed.KRadiuus) // TODO: fix
  for i := 0; i < len(kSignValues); i++ {
    if compressed.KSignBits.Test(uint(i)) {
      kSignValues[i] = 1.0
    } else {
      kSignValues[i] = -1.0
    }
  }

  // Pseudo-inverse of G to recover error approximation
  kErrorRecovered := pseudoInverse(qjl.G, kSignValues)

  // Dequantize + add correction
  kDequant := dequantize(compressed.KPacked, ...)
  kCorrected := add(kDequant, kErrorRecovered)

  // Denormalize and un-rotate
  kOriginal := matmul(qjl.R.T(), kCorrected)

  return kOriginal, vOriginal, nil
}

func generateGaussianMatrix(seed int64, headDim int) *mat.Dense {
  m := headDim / 2
  data := make([]float64, m*headDim)
  rng := rand.New(rand.NewSource(seed))
  for i := range data {
    data[i] = rng.NormFloat64()
  }
  return mat.NewDense(m, headDim, data)
}
```

---

<a name="cli"></a>

## 8. CLI & Resposta (Bubbletea + Glamour)

### 8.1 Chat Command

```go
// cmd/vectora/cmd_chat.go

func chatCmd(cfg *config.Config) *cobra.Command {
  return &cobra.Command{
    Use:   "chat",
    Short: "Interactive chat session",
    RunE: func(cmd *cobra.Command, args []string) error {
      // Initialize
      kv, _ := storage.NewBoltDB(cfg.DataDir)
      vec, _ := storage.NewUsearchIndex(cfg.DataDir)
      v, _ := core.NewVectora(cfg, kv, vec)

      // Start bubbletea app
      p := tea.NewProgram(NewChatModel(v, cfg))
      return p.Run()
    },
  }
}

// ChatModel: bubbletea model
type ChatModel struct {
  core     *core.Vectora
  messages []Message
  input    string
  streaming bool
}

type Message struct {
  Role    string // "user" ou "assistant"
  Content string
}

func (m ChatModel) Init() tea.Cmd {
  return nil
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c":
      return m, tea.Quit
    case "enter":
      userMsg := m.input
      m.input = ""
      m.messages = append(m.messages, Message{"user", userMsg})

      // Stream response
      return m, m.streamResponse(userMsg)
    default:
      m.input += msg.String()
    }

  case ResponseToken:
    m.messages[len(m.messages)-1].Content += msg.Token
  }

  return m, nil
}

func (m ChatModel) View() string {
  var s string

  // Display messages
  s += "┌─ Chat History ─────────────────────┐\n"
  for _, msg := range m.messages {
    if msg.Role == "user" {
      s += fmt.Sprintf("┃ You: %s\n", msg.Content)
    } else {
      s += fmt.Sprintf("┃ Vectora: %s\n", msg.Content)
    }
  }
  s += "└─────────────────────────────────────┘\n"

  // Input field
  s += "\nYour message:\n"
  s += m.input + " ▌\n"
  s += "(Press Ctrl+C to exit)\n"

  return s
}

func (m ChatModel) streamResponse(userMsg string) tea.Cmd {
  return func() tea.Msg {
    ctx := context.Background()
    respChan, _ := m.core.Chat(ctx, "local_user", userMsg)

    m.messages = append(m.messages, Message{"assistant", ""})

    for token := range respChan {
      // Return each token as event
      return ResponseToken{token}
    }

    return nil
  }
}

type ResponseToken struct {
  Token string
}
```

### 8.2 Plan Mode (Visualization)

```go
// internal/server/plan_display.go

func (s *Server) displayPlan(w http.ResponseWriter, plan ExecutionPlan) {
  fmt.Fprintf(w, "\n")
  fmt.Fprintf(w, "╔════════════════════════════════════════════════╗\n")
  fmt.Fprintf(w, "║ EXECUTION PLAN                                 ║\n")
  fmt.Fprintf(w, "╚════════════════════════════════════════════════╝\n")

  for i, step := range plan.Steps {
    status := "◯"  // pending
    fmt.Fprintf(w, "%d. [%s] %s\n", i+1, status, step.Tool)
    fmt.Fprintf(w, "   Args: %v\n", step.Args)
    if len(step.DependsOn) > 0 {
      fmt.Fprintf(w, "   Depends on: %v\n", step.DependsOn)
    }
  }

  fmt.Fprintf(w, "\nExecute? [y/N/edit] ")
}
```

---

<a name="mcp"></a>

## 9. MCP Protocol Integration

### 9.1 MCP Server (Go)

```go
// internal/mcp/server.go

package mcp

import (
  "context"
  "encoding/json"
  "fmt"

  mcpgo "github.com/mark3labs/mcp-go"
)

type MCPServer struct {
  core *core.Vectora
  server *mcpgo.MCPServer
}

func NewMCPServer(core *core.Vectora) *MCPServer {
  return &MCPServer{
    core: core,
    server: mcpgo.NewMCPServer(),
  }
}

func (m *MCPServer) RegisterTools() {
  // Vectora/rag_search tool
  m.server.AddTool(&mcpgo.Tool{
    Name:        "vectora/rag_search",
    Description: "Search embeddings by semantic similarity",
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "query": map[string]interface{}{"type": "string"},
        "top_k": map[string]interface{}{"type": "integer", "default": 5},
      },
      "required": []string{"query"},
    },
  }, func(args map[string]interface{}) (interface{}, error) {
    query := args["query"].(string)
    topK := int(args["top_k"].(float64))

    results, err := m.core.vec.Search(context.Background(), query, topK)
    if err != nil {
      return nil, err
    }

    return results, nil
  })

  // Vectora/edit_file tool
  m.server.AddTool(&mcpgo.Tool{
    Name:        "vectora/edit_file",
    Description: "Edit a file with precise ranges",
    InputSchema: map[string]interface{}{
      "type": "object",
      "properties": map[string]interface{}{
        "path": map[string]interface{}{"type": "string"},
        "start_line": map[string]interface{}{"type": "integer"},
        "end_line": map[string]interface{}{"type": "integer"},
        "new_content": map[string]interface{}{"type": "string"},
      },
    },
  }, func(args map[string]interface{}) (interface{}, error) {
    // Implementação via tools.Registry
    return nil, nil
  })
}

func (m *MCPServer) Start(addr string) error {
  m.RegisterTools()
  return m.server.ListenAndServe(addr)
}
```

---

<a name="agent"></a>

## 10. Agent Principal Mode (Detalhado)

Já implementado em seção 4.2 (Agent Orchestrator).

**Fluxo resumido:**

1. User: `vectora agent "refatore meu código"`
2. Intent classifier: confidence < 0.7 → vai para agent
3. Planning phase: LLM gera execution plan JSON
4. Execution phase: tools são invocadas em paralelo respeitando dependências
5. Reflection phase (opcional): LLM avalia se objetivo foi alcançado
6. Response: concatena resultados + LLM final summary

---

<a name="harness"></a>

## 11. Sistema de Treinamento (Harness)

### 11.1 Harness YAML Format

```yaml
# ~/.vectora/harness/golang_refactor.yaml

harness:
  name: "Go Code Refactoring"
  description: "Test refactoring suggestions for Go code"
  version: 1

cases:
  - id: "case_001"
    name: "Add error wrapping"
    input:
      task: "Refactor this Go function to add proper error wrapping"
      file_content: |
        func readFile(path string) ([]byte, error) {
            data, err := os.ReadFile(path)
            if err != nil {
                return nil, err
            }
            return data, nil
        }
    expected:
      tool_calls:
        ["analyze_code_patterns", "refactor_with_context", "write_file"]
      quality_score: 0.9 # LLM evaluates if refactoring is good
      contains_patterns: ["fmt.Errorf", "wrapping"]

  - id: "case_002"
    name: "Extract interface"
    input:
      task: "Extract an interface from this struct"
      file_content: "... código do caso ..."
    expected:
      tool_calls: ["analyze_code_patterns"]
      quality_score: 0.85
```

### 11.2 Harness Runner (Go)

```go
// cmd/vectora/cmd_harness.go

func harnessCmd(cfg *config.Config) *cobra.Command {
  return &cobra.Command{
    Use:   "harness run [--file harness.yaml]",
    Short: "Run harness test suite",
    RunE: func(cmd *cobra.Command, args []string) error {
      harnessFile := "~/.vectora/harness/*.yaml"

      var results []HarnessResult

      // Carregar todos os harness files
      harnessFiles, _ := filepath.Glob(harnessFile)

      for _, file := range harnessFiles {
        harness, _ := loadHarness(file)

        for _, tc := range harness.Cases {
          // Executar caso
          result, _ := v.Agent(ctx, "harness", tc.Input.Task)

          // Validar resultado
          score := evaluateResult(result, tc.Expected)

          results = append(results, HarnessResult{
            CaseID: tc.ID,
            Score:  score,
            Passed: score >= 0.8,
          })
        }
      }

      // Reportar
      fmt.Printf("Harness Results: %d/%d passed\n", passCount(results), len(results))
      for _, r := range results {
        status := "✓"
        if !r.Passed {
          status = "✗"
        }
        fmt.Printf("%s %s: %.2f\n", status, r.CaseID, r.Score)
      }

      return nil
    },
  }
}
```

---

<a name="segurança"></a>

## 12. Segurança

### 12.1 Sanitizer Tool

```go
// internal/safety/sanitizer.go

package safety

import (
  "regexp"
)

var secretPatterns = []*regexp.Regexp{
  regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),                    // OpenAI key
  regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                        // AWS access key
  regexp.MustCompile(`ghp_[A-Za-z0-9_]{36,255}`),               // GitHub token
  regexp.MustCompile(`-----BEGIN (RSA|EC|PGP) PRIVATE KEY`),    // Private keys
}

type Sanitizer struct{}

func (s *Sanitizer) ScanContent(content string) []SecretMatch {
  var matches []SecretMatch

  for i, pattern := range secretPatterns {
    results := pattern.FindAllStringIndex(content, -1)
    for _, r := range results {
      matches = append(matches, SecretMatch{
        Pattern: pattern.String(),
        Line:    countLines(content[:r[0]]),
        Start:   r[0],
        End:     r[1],
      })
    }
  }

  return matches
}

type SecretMatch struct {
  Pattern string
  Line    int
  Start   int
  End     int
}
```

### 12.2 .vectoraignore / .embedignore Parser

```go
// internal/safety/ignore.go

package safety

import (
  "github.com/monochromegane/go-gitignore"
)

type IgnoreRules struct {
  vectoraIgnore *gitignore.GitIgnore
  embedIgnore   *gitignore.GitIgnore
}

func (ir *IgnoreRules) ShouldIgnoreForLLM(path string) bool {
  return ir.vectoraIgnore.Match(path, false)
}

func (ir *IgnoreRules) ShouldIgnoreForEmbedding(path string) bool {
  return ir.embedIgnore.Match(path, false)
}

func LoadIgnoreRules(projectRoot string) (*IgnoreRules, error) {
  vIgnore, err := gitignore.NewGitIgnore(projectRoot + "/.vectoraignore")
  if err != nil {
    // Default if file not found
    vIgnore = &gitignore.GitIgnore{}
  }

  eIgnore, err := gitignore.NewGitIgnore(projectRoot + "/.embedignore")
  if err != nil {
    eIgnore = &gitignore.GitIgnore{}
  }

  return &IgnoreRules{
    vectoraIgnore: vIgnore,
    embedIgnore:   eIgnore,
  }, nil
}
```

### 12.3 Tool Sandbox

```go
// internal/tools/executor.go

func (r *Registry) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
  toolDef, ok := r.tools[toolName]
  if !ok {
    return nil, fmt.Errorf("tool not found: %s", toolName)
  }

  // 1. Validate input against schema
  if err := validate(args, toolDef.InputSchema); err != nil {
    return nil, fmt.Errorf("validation: %w", err)
  }

  // 2. Check rate limiting
  if r.limiter.Take() == false {
    return nil, fmt.Errorf("rate limit exceeded")
  }

  // 3. Create timeout context
  ctx, cancel := context.WithTimeout(ctx, toolDef.Timeout)
  defer cancel()

  // 4. Execute with sandbox
  var result interface{}
  var err error

  if toolDef.Type == "native" {
    result, err = toolDef.Handler(args)
  } else if toolDef.Type == "subprocess" {
    result, err = r.execSubprocess(ctx, toolDef, args)
  }

  // 5. Log execution
  r.logAudit(toolName, args, result, err)

  return result, err
}

func (r *Registry) execSubprocess(ctx context.Context, tool *ToolDef, args map[string]interface{}) (interface{}, error) {
  // Whitelist: permitir apenas comandos seguros
  whitelisted := []string{"go", "cargo", "npm", "git", "python"}

  cmd := args["command"].(string)

  // Rejeitar comandos perigosos
  dangerousCommands := []string{"rm", "dd", "sudo", "chmod", "chown"}
  for _, danger := range dangerousCommands {
    if strings.Contains(cmd, danger) {
      return nil, fmt.Errorf("command not allowed: %s", danger)
    }
  }

  // Executar com timeout
  output, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()

  return string(output), err
}
```

---

<a name="git"></a>

## 13. Git/GitHub Integration

### 13.1 Git Operations

```go
// internal/git/repo.go

package git

import (
  "github.com/go-git/go-git/v5"
  "github.com/go-git/go-git/v5/plumbing"
)

type Repo struct {
  r *git.Repository
}

func Open(path string) (*Repo, error) {
  r, err := git.PlainOpen(path)
  return &Repo{r}, err
}

func (r *Repo) GetRemoteURL() (string, error) {
  cfg, _ := r.r.Config()
  if origin, ok := cfg.Remotes["origin"]; ok {
    if len(origin.URLs) > 0 {
      return origin.URLs[0], nil
    }
  }
  return "", fmt.Errorf("no remote found")
}

func (r *Repo) CreateBranch(name string) error {
  headRef, _ := r.r.Head()
  ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+name), headRef.Hash())
  return r.r.Storer.SetReference(ref)
}

func (r *Repo) CommitChanges(message string) (string, error) {
  w, _ := r.r.Worktree()
  w.Add(".")

  commit, err := w.Commit(message, &git.CommitOptions{
    Author: &object.Signature{
      Name:  "Vectora AI",
      Email: "vectora@kaffyn.dev",
      When:  time.Now(),
    },
  })

  return commit.String(), err
}
```

### 13.2 GitHub Integration

```go
// internal/git/github.go

func (r *Repo) CreatePullRequest(title, body, baseBranch string) (int, error) {
  client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

  req := &github.NewPullRequest{
    Title:               github.String(title),
    Body:                github.String(body),
    Head:                github.String(r.currentBranch),
    Base:                github.String(baseBranch),
    MaintainerCanModify: github.Bool(true),
  }

  pr, _, err := client.PullRequests.Create(context.Background(), owner, repo, req)
  return pr.GetNumber(), err
}
```

---

<a name="telemetry"></a>

## 14. Telemetry & Observability

### 14.1 Metrics Collection

```go
// internal/telemetry/metrics.go

package telemetry

import (
  "go.opentelemetry.io/otel"
  "go.opentelemetry.io/otel/metric"
)

var (
  meter = otel.Meter("vectora")

  // Tool metrics
  toolInvocations, _ = meter.Int64Counter("tool.invocations")
  toolLatency, _     = meter.Float64Histogram("tool.latency_ms")

  // Agent metrics
  agentSteps, _      = meter.Int64Counter("agent.steps_completed")
  agentLatency, _    = meter.Float64Histogram("agent.latency_ms")

  // LLM metrics
  llmTokens, _       = meter.Int64Counter("llm.tokens_used")
  llmLatency, _      = meter.Float64Histogram("llm.latency_ms")
)

func RecordToolExecution(toolName string, latency float64) {
  ctx := context.Background()
  toolInvocations.Add(ctx, 1, metric.WithAttributes(
    attribute.String("tool", toolName),
  ))
  toolLatency.Record(ctx, latency)
}
```

### 14.2 Structured Logging (zerolog)

```go
// internal/telemetry/logs.go

import "github.com/rs/zerolog"

func InitLogging(cfg *config.Config) {
  // JSON structured logging
  zerolog.TimeFieldFormat = time.RFC3339Nano

  logFile, _ := os.OpenFile(
    cfg.DataDir+"/logs/vectora.log",
    os.O_CREATE|os.O_APPEND|os.O_WRONLY,
    0644,
  )

  log.Logger = zerolog.New(logFile).With().Timestamp().Logger()
}

// Usage
log.Info().
  Str("tool", "rag_search").
  Int("results", 5).
  Float32("latency_ms", 234.5).
  Msg("tool execution completed")
```

---

<a name="multiplatform"></a>

## 15. Multi-OS Support

### 15.1 Build Script

```bash
#!/bin/bash
# scripts/build.sh

set -e

VERSION=$(git describe --tags --always)
LDFLAGS="-X github.com/kaffyn/vectora/internal/config.Version=$VERSION"

echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/vectora-darwin-arm64 ./cmd/vectora

echo "Building for macOS (x86_64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/vectora-darwin-amd64 ./cmd/vectora

echo "Building for Linux (x86_64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/vectora-linux-amd64 ./cmd/vectora

echo "Building for Windows (x86_64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/vectora-windows-amd64.exe ./cmd/vectora

echo "Creating universal macOS binary..."
lipo -create dist/vectora-darwin-arm64 dist/vectora-darwin-amd64 -output dist/vectora-darwin-universal

echo "Done. Binaries in dist/"
```

### 15.2 Platform-Specific Code

```go
// internal/config/paths.go

//go:build darwin
// +build darwin

package config

import "os"

func getDataDir() string {
  return os.ExpandEnv("$HOME/Library/Application Support/Vectora")
}

---

//go:build linux
// +build linux

package config

import "os"

func getDataDir() string {
  return os.ExpandEnv("$HOME/.local/share/vectora")
}

---

//go:build windows
// +build windows

package config

import "os"

func getDataDir() string {
  return os.ExpandEnv("%APPDATA%\\Vectora")
}
```

---

<a name="deployment"></a>

## 16. Deployment & DevOps

### 16.1 Docker Multistage Build

```dockerfile
# Dockerfile (produces ~20MB final image)

FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.* .
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o vectora ./cmd/vectora

---

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/vectora /usr/local/bin/

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/vectora", "serve"]
```

### 16.2 docker-compose.yml

```yaml
version: "3.9"

services:
  vectora:
    build: .
    ports:
      - "8080:8080"
    environment:
      - VECTORA_PROVIDER=anthropic
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    volumes:
      - vectora-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  vectora-data:
```

### 16.3 Kubernetes Deployment

```yaml
# k8s/deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: vectora-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: vectora
  template:
    metadata:
      labels:
        app: vectora
    spec:
      containers:
        - name: vectora
          image: kaffyn/vectora:latest
          ports:
            - containerPort: 8080
          env:
            - name: VECTORA_PROVIDER
              value: anthropic
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: vectora-secrets
                  key: anthropic-key
          resources:
            requests:
              memory: "512Mi"
              cpu: "500m"
            limits:
              memory: "2Gi"
              cpu: "2000m"
          livenessProbe:
            httpGet:
              path: /api/v1/health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
```

---

## Resumo: Proposta D (Go Microkernel)

| Aspecto                | Implementação                                                      |
| ---------------------- | ------------------------------------------------------------------ |
| **Core Daemon**        | Binário Go + BoltDB local + usearch                                |
| **Service (Cloud)**    | Mesmo binário, modo `--server`, JWT+RBAC, multi-tenant namespacing |
| **Companion**          | Extensão VSCode TypeScript, MCP stdio/WebSocket                    |
| **TurboQuant**         | Implementado em Go puro (gonum, willf/bitset)                      |
| **CLI**                | cobra + bubbletea + glamour (markdown rendering)                   |
| **Segurança**          | .vectoraignore/.embedignore, tool sandbox, audit log               |
| **Git**                | go-git (local), GitHub API (PR automation)                         |
| **Telemetry**          | OpenTelemetry + zerolog (structured JSON)                          |
| **Deployment**         | Docker < 20MB, Kubernetes-ready, cross-platform                    |
| **Developer Velocity** | Rápido: SDKs oficiais, single binary, sem runtime                  |
| **Escalabilidade**     | goroutines nativas → 10k+ concurrent users                         |

---

**Pronto para implementação. Comece pelo Core Daemon, depois Service, Companion por último.**
