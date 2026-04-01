// EventBus — central event dispatcher
// Known issues: no retry logic, no dead-letter queue, sync handlers block

import { Event, EventHandler, EventPayload, Subscription } from "./types";

type HandlerMap = Map<string, Set<EventHandler>>;

export class EventBus {
  private handlers: HandlerMap = new Map();
  private history: Event[] = [];

  subscribe<T extends EventPayload>(
    eventType: string,
    handler: EventHandler<T>
  ): Subscription {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    // BUG: cast loses type safety
    this.handlers.get(eventType)!.add(handler as EventHandler);

    return {
      unsubscribe: () => {
        this.handlers.get(eventType)?.delete(handler as EventHandler);
      },
    };
  }

  // publish fires handlers — no error isolation (one failure breaks all)
  publish<T extends EventPayload>(event: Event<T>): void {
    this.history.push(event as Event);
    const handlers = this.handlers.get(event.type);
    if (!handlers) return;

    handlers.forEach((handler) => {
      // ISSUE: async handlers not awaited — fire-and-forget
      handler(event as Event);
    });
  }

  // emit is shorthand — builds Event and calls publish
  emit(type: string, payload: EventPayload, source?: string): void {
    const event: Event = {
      id: Math.random().toString(36).slice(2), // ISSUE: not UUID
      type,
      payload,
      timestamp: new Date(),
      source,
    };
    this.publish(event);
  }

  getHistory(): Event[] {
    return this.history;
  }

  clearHandlers(): void {
    this.handlers.clear();
  }
}

// Singleton global bus — makes testing harder
export const globalBus = new EventBus();
