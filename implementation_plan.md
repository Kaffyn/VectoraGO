# Refatoração Arquitetural do Vectora — Go Microkernel

## Context

O Vectora tomou decisões arquiteturais imaturas:

1. **chromemgo** ao invés de **usearch-go HNSW** — sem suporte a turboquant, sem escala
2. **Processamento sequencial** — sem goroutines, sem paralelismo, sem concorrência real
3. **Sem containerização** — sem Docker, Kubernetes, sem cloud-native readiness
4. **Sem harness** — sem test infrastructure, sem garantias de qualidade
5. **Tratado como projeto acadêmico** — precisa ser endurecido como projeto empresarial

O arquivo `vectora.md` detalha a arquitetura correta: **3 camadas** (Core Daemon + Service Cloud-Native + VSCode Extension), **stack maduro** (usearch-go, BoltDB, TurboQuant, OpenTelemetry, Kubernetes), e **harness system** para testes sistematizados.

**Necessidade:** Refatoração fundamental do Vectora seguindo a Proposta D. Não é incremento — é redesign completo com nova estrutura de código.

---

## Fases de Refatoração (Proposta D)

### FASE 1: Fundação — Vector Storage + Quantização

**Objetivo:** Substituir chromemgo por usearch-go HNSW com TurboQuant

**Tarefas:**

1. **Migração usearch-go**
   - Remover `github.com/philippgille/chromem-go` de `go.mod`
   - Adicionar `github.com/unum-cloud/usearch-go` (v0.2.0+)
   - Refatorar `core/db/vector.go` → `internal/storage/vector.go` (nova estrutura)
   - Implementar abstração HNSW com `AddVector()`, `SearchVector()`
   - Suporte a 6+ métricas (cosine, L2, IP, dot, custom)

2. **TurboQuant Implementation**
   - `internal/quant/turboquant.go` (orquestrador)
   - `internal/quant/polar.go` (PolarQuant stage: rotation + angle extraction)
   - `internal/quant/qjl.go` (QJL correction: 1-bit bias stabilization)
   - `internal/quant/codebook.go` (Lloyd-Max codebooks + quantization)
   - Teste com KV cache compression (embedding → 150-200 bytes)

3. **Storage Layer Tests**
   - Unit tests: usearch initialization, add, search, serialize
   - Integration: TurboQuant compression roundtrip (encode/decode)

**Deliverable:** `internal/storage/` refatorado, zero chromemgo references

---

### FASE 2: Restructuring — Reorganizar para `internal/` Layout

**Objetivo:** Seguir Proposta D structure, preparar para cloud-native

**Tarefas:**

1. **Directory restructure**

   ```
   core/ → internal/
   core/llm/ → internal/llm/
   core/tools/ → internal/tools/
   core/tray/ → (delete, move config to internal/config/)
   core/infra/ → internal/storage/ (unificar KV + Vector)
   core/manager/ → internal/manager/
   core/engine/ → internal/core/agent.go, planner.go, etc.
   core/api/ → internal/api/ (para daemon local)

   NEW:
   internal/server/ → HTTP server (Service mode)
   internal/auth/ → JWT + RBAC
   internal/mcp/ → MCP server impl (já existe parcialmente)
   internal/config/ → Config loading
   internal/telemetry/ → OpenTelemetry + zerolog
   internal/safety/ → Sanitizer + Sandbox
   ```

2. **Import updates**
   - Go: buscar/replace todos `core/` → `internal/` (exceto cmd/)
   - Preservar APIs públicas em `pkg/vectora/` (Go SDK)
   - **IMPORTANTE:** tray module é crítico para CLI/local mode - reorganizar em `internal/tray/`, NOT remover

3. **Dependency cleanup**
   - Consolidar vendor deps em `go.mod` (documentar cada um)
   - manter tray.ReloadActiveProvider() call em internal/api/ipc/router.go provider.reload handler

**Deliverable:** Codebase reorganizado, sem circular imports, tudo em `internal/`

---

### FASE 3: Concorrência — Goroutines + Channels

**Objetivo:** Implementar paralelismo real (não mais serial)

**Tarefas:**

1. **Embedding Parallelization**
   - `internal/core/embedder.go`: refatorar loop sequencial
   - Implementar worker pool (4-8 goroutines)
   - Use `sync.WaitGroup` para fan-out/fan-in chunks
   - Each worker: chunk → embed → upsert (atomic)

2. **Tool Execution Pipeline**
   - `internal/tools/executor.go`: parallelizar tool calls
   - Executar múltiplas tools concorrentemente (dependências respeitadas)
   - Channel de results: `toolResults chan ToolResult`

3. **Resource Pool + Rate Limiting**
   - `internal/manager/resource_pool.go`: já existe, refatorar
   - Semáforos por tenant (não global)
   - Circuit breaker com backoff exponencial (missing)
   - Timeout automático para LLM calls

4. **Streaming Responses**
   - `internal/core/agent.go`: streaming de respostas via channel
   - HTTP/WebSocket handler: range over channel, escrever ao cliente

**Deliverable:** Embeddings paralelos, tool calls concorrentes, resource pool endurecido

---

### FASE 4: Cloud-Native — HTTP Server + Auth

**Objetivo:** Implementar Service mode (multi-tenant, HTTP API)

**Tarefas:**

1. **HTTP Server (chi router)**
   - `internal/server/http.go`: chi router setup
   - Routes:
     - `POST /api/v1/chat` (streaming JSON)
     - `POST /api/v1/plan` (planning endpoint)
     - `POST /api/v1/embed` (batch embedding)
     - `GET /api/v1/memory` (memory retrieval)
     - `GET /api/v1/health` (liveness)
     - `WebSocket /ws/agent` (streaming agent responses)

2. **JWT Auth + RBAC**
   - `internal/auth/jwt.go`: issue/validate JWT tokens
   - `internal/auth/rbac.go`: role-based access control
   - Middleware: extract user_id, roles from token
   - Enforce permissions per endpoint

3. **Multi-tenant KV Namespacing**
   - BoltDB: `buckets: /user/{user_id}/...`
   - Per-user isolation via middleware
   - Quota enforcement (storage, API calls)

4. **Telemetry**
   - `internal/telemetry/metrics.go`: OpenTelemetry metrics
   - `internal/telemetry/traces.go`: OTLP exporter
   - `internal/telemetry/logs.go`: zerolog structured logging
   - Export to Datadog/Prometheus/Jaeger

**Deliverable:** HTTP Service pronto para Kubernetes, multi-tenant isolado

---

### FASE 5: DevOps — Docker + Kubernetes

**Objetivo:** Containerizar, deployar em Kubernetes

**Tarefas:**

1. **Dockerfile Multi-stage**
   - Build stage: Go 1.23, build com CGO (usearch precisa)
   - Final stage: Alpine, ~20MB final image
   - Healthcheck: `/api/v1/health` probe

2. **docker-compose.yml**
   - Vectora service
   - Environment: API keys, JWT_SECRET
   - Volume: `/data` para persistência BoltDB
   - Port: 8080

3. **Kubernetes Manifests** (`k8s/`)
   - `deployment.yaml`: 3 replicas, resource limits
   - `service.yaml`: ClusterIP, port 8080
   - `configmap.yaml`: application config
   - `secret.yaml`: API keys (from env vars)
   - `hpa.yaml`: autoscaling (target: 70% CPU)
   - `ingress.yaml`: TLS, domain routing

4. **CI/CD Pipeline Updates**
   - Build Docker image em cada commit
   - Push a registry (Docker Hub, ECR, etc.)
   - Deploy ao k8s via CD (ArgoCD, FluxCD)

**Deliverable:** Kubernetes-ready, TLS, autoscaling, observability

---

### FASE 6: Harness System

**Objetivo:** Test infrastructure para garantia de qualidade

**Tarefas:**

1. **Harness YAML Format**
   - `~/.vectora/harness/*.yaml`: test case definitions
   - Fields: task input, expected tool calls, quality score
   - Example: Go refactoring, document generation, code analysis

2. **Harness Runner**
   - `cmd/vectora/cmd_harness.go`: CLI command
   - Load all harness files, run cases in parallel
   - Evaluate outputs: tool calls check, LLM quality scoring
   - Report: pass/fail, score, timing

3. **Test Harness Library**
   - Built-in harness suites: coding, documentation, analysis
   - User-defined harness support
   - Metrics: pass rate, avg score, timing

**Deliverable:** `vectora harness run` command, 50+ test cases built-in

---

### FASE 7: VSCode Extension + CLI

**Objetivo:** Companion extension, polished CLI

**Tarefas:**

1. **VSCode Extension Refactor**
   - Uso de MCP stdio (já existe)
   - Connect to Service mode (WebSocket) como fallback
   - Chat panel, memory browser, plan viewer (webviews)

2. **CLI Enhancements**
   - Cobra commands: `chat`, `ask`, `embed`, `plan`, `harness`
   - Bubbletea TUI: interactive chat, plan visualization
   - Glamour: markdown rendering para respostas

3. **Config Management**
   - YAML config file
   - Provider selection (anthropic, openai, gemini)
   - Model selection, embedding model
   - API key management

**Deliverable:** Polished CLI + VSCode extension, ready for release

---

### FASE 8: Documentation + Release

**Objetivo:** Documentação completa, release 1.0

**Tarefas:**

1. **Architecture Documentation**
   - ARCHITECTURE.md (detailed)
   - API.md (REST endpoints, examples)
   - DEPLOYMENT.md (Docker, Kubernetes)
   - HARNESS.md (test harness guide)

2. **User Guide**
   - Quick start
   - CLI reference
   - VSCode extension guide
   - Service deployment guide

3. **Release**
   - Version bump to 1.0.0
   - GitHub Release notes
   - Docker image push
   - NPM package publication (extensions)

**Deliverable:** Fully documented, production-ready v1.0

---

## Cronograma (Estimado)

| Fase      | Descrição               | Semanas           | Parallelizável              |
| --------- | ----------------------- | ----------------- | --------------------------- |
| **1**     | usearch-go + TurboQuant | 2-3               | Não (dependência)           |
| **2**     | Restructuring           | 1-2               | Parcial                     |
| **3**     | Goroutines + Channels   | 1-2               | Sim (paralelo com fase 2)   |
| **4**     | HTTP Server + Auth      | 2-3               | Sim (paralelo com fase 2-3) |
| **5**     | Docker + Kubernetes     | 1-2               | Sim (paralelo com fase 4)   |
| **6**     | Harness System          | 1-2               | Sim (paralelo com fase 5)   |
| **7**     | VSCode + CLI Polish     | 1-2               | Sim                         |
| **8**     | Documentation + Release | 1                 | Final                       |
| **TOTAL** |                         | **10-14 semanas** | com paralelização           |

---

## Prioridades Críticas

1. **Phase 1 (usearch-go)** — nada funciona sem vector storage correto
2. **Phase 3 (Goroutines)** — sem concorrência real, cloud-native não escala
3. **Phase 6 (Harness)** — sem testes sistematizados, não há garantia de qualidade
4. Rest: pode ser paralelo

---

## Referências Críticas

- `vectora.md` — Proposta D completa (todas as seções 1-16)
- `usearch-vs-all-storage-embedding.md` — por que usearch > chromemgo
- `hnsw-vs-chromemgo-turboquant-efficiency.md` — TurboQuant requisitos
- Código atual: `core/` (será refatorado para `internal/`)

---

## Verification

```bash
# Fase 1: Vector storage
go test ./internal/storage/... -v
go test ./internal/quant/... -v

# Fase 3: Concorrência
go test ./internal/core/... -v -race

# Fase 4: HTTP
curl http://localhost:8080/api/v1/health

# Fase 5: Kubernetes
kubectl apply -f k8s/
kubectl get pods -w

# Fase 6: Harness
vectora harness run

# Fase 7: CLI/Extension
vectora chat "what is 2+2?"
# VSCode: Ctrl+Shift+P → Vectora: Chat

# Fase 8: Release
git tag v1.0.0
gh release create v1.0.0
```
