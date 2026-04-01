# Architecture: Event-Driven System

## Overview

This project implements event-driven architecture using a central EventBus.

```
Producer ──emit()──► EventBus ──dispatch──► Consumer A
                               └──dispatch──► Consumer B
                               └──dispatch──► Consumer C
```

## Components

### EventBus
Central dispatcher. Maintains handler registry (Map<eventType, Set<Handler>>).

### BaseListener
Abstract listener with retry logic. Concrete implementations override `handler()`.

### Event Types
Typed domain events with structured payloads. See `src/events/types.ts`.

## Known Limitations

1. **No async coordination** — publish() does not await async handlers
2. **Global singleton** — `globalBus` makes unit testing difficult
3. **In-memory state** — workflow state lost on restart
4. **No dead-letter queue** — failed events after max retries are silently dropped
5. **Non-UUID IDs** — `Math.random().toString(36)` collisions possible under load
6. **No backpressure** — unlimited concurrent handlers
