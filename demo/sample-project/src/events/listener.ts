// Abstract listener with basic retry logic
import { Event, EventHandler, EventPayload, Subscription } from "./types";
import { EventBus } from "./bus";

export abstract class BaseListener<T extends EventPayload> {
  abstract eventType: string;
  abstract handler(event: Event<T>): Promise<void>;

  private subscription?: Subscription;
  private retries = 0;
  private maxRetries = 3; // ISSUE: hardcoded, not configurable

  listen(bus: EventBus): void {
    this.subscription = bus.subscribe<T>(
      this.eventType,
      this.handleWithRetry.bind(this) as EventHandler<T>
    );
  }

  // Retry logic — exponential backoff but no jitter
  private async handleWithRetry(event: Event<T>): Promise<void> {
    try {
      await this.handler(event);
      this.retries = 0;
    } catch (err) {
      this.retries++;
      if (this.retries <= this.maxRetries) {
        const delay = Math.pow(2, this.retries) * 100; // ISSUE: no jitter
        await new Promise((r) => setTimeout(r, delay));
        return this.handleWithRetry(event);
      }
      // ISSUE: errors silently swallowed after max retries
      console.error(
        `[${this.eventType}] Handler failed after ${this.maxRetries} retries:`,
        err
      );
      this.retries = 0;
    }
  }

  stop(): void {
    this.subscription?.unsubscribe();
  }
}
