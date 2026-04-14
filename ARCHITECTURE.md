# Vectora Architecture

Vectora is a local-first, agentic AI coding assistant designed for enterprise-grade security and developer productivity. This document details the 3-tier microkernel architecture.

## 1. System Overview

Vectora consists of three primary layers:

- **Core Daemon (Go)**: The brain of the system. Handles LLM orchestration, vector storage (usearch), and tool execution.
- **Service Layer (Cloud-Native)**: A multi-tenant HTTP/WebSocket service that allows remote use and team collaboration.
- **Companion (VS Code Extension)**: The primary UI, connecting to the Core via the ACP (Agent Client Protocol).

## 2. Core Kernel Design

The Core is built as a **Go Microkernel** with a focus on:

- **Concurrency**: Native goroutines for parallel embedding and tool execution.
- **Security**: The `Guardian` middleware enforces file system access policies and prevents path traversal.
- **Efficiency**: Bit-quantized vectors (TurboQuant) reduce memory footprint by 10x.

### Subsystems:

- `internal/core`: Agent logic, planner, and task orchestration.
- `internal/storage`: Hybrid DB (BoltDB for KV, usearch for HNSW vectors).
- `internal/llm`: Provider-agnostic router (Gemini, Claude, OpenAI).
- `internal/tools`: Registry of sandboxed tools (read_file, write_file, web_search).
- `internal/harness`: Automated quality validation engine.

## 3. Communication Protocols

Vectora uses three main protocols:

1. **ACP (Agent Client Protocol)**: JSON-RPC 2.0 over stdio/pipes. Used between VS Code and the local Core.
2. **MCP (Model Context Protocol)**: Standard protocol for tool discovery and resource mapping.
3. **REST/WS**: Standard web protocols for the Service mode.

## 4. Multi-Tenancy & Isolation

In Service mode, the `TenantManager` provides strict isolation:

- **Namespaced Storage**: Each user has their own BoltDB buckets and vector indexes.
- **RLS (Row Level Security)**: Enforced at the middleware level.
- **Resource Quotas**: Token limits and rate limiting per resident tenant.

## 5. Quality Assurance (The Harness)

The **Vectora Harness** evaluates agent performance across 3 dimensions:

1. **Retrieval**: Measuring MRR@5 for RAG accuracy.
2. **Reasoning**: Validating tool call sequences using strict and unordered assertions.
3. **Execution**: LLM-as-a-Judge grading of final outputs against rubrics (Correctness, Security, etc).

---

_Vectora Specification v1.0.0 (April 2026)_
