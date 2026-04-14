# Vectora REST API Reference

Vectora Service exposes a RESTful API for chat, planning, and knowledge management. All endpoints (except health) require a JWT token.

## Authentication

Bearer token authentication: `Authorization: Bearer <JWT>`

## Endpoints

### 1. Chat & Query

#### `POST /api/v1/chat`

Standard request/response chat.

- **Body**: `{"query": "string", "conversation_id": "string", "model": "string"}`
- **Response**: `{"answer": "string", "tokens": 123}`

#### `POST /api/v1/chat/stream`

Streaming chat (Server-Sent Events).

- **Body**: Same as above.
- **Output**: Stream of JSON tokens.

### 2. Planning & Agentic Mode

#### `POST /api/v1/plan`

Generates an execution plan for a complex task.

- **Body**: `{"task": "string"}`
- **Response**: `{"plan_id": "uuid", "steps": [...]}`

### 3. Knowledge Base (RAG)

#### `POST /api/v1/embed/start`

Starts a background indexing job.

- **Body**: `{"path": "string", "include": "*.go"}`
- **Response**: `{"job_id": "uuid"}`

#### `POST /api/v1/embed/search`

Semantic search across the indexes.

- **Body**: `{"query": "string", "limit": 5}`
- **Response**: `{"results": [...]}`

### 4. System

#### `GET /api/v1/health`

Liveness check (Public).

- **Response**: `{"status": "healthy"}`

#### `GET /api/v1/ws/agent` (WebSocket)

Full bidirectional Agent session. Protocol: JSON-RPC 2.0.

---

_Generated: April 2026_
