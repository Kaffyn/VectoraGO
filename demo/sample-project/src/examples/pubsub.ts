// Pub/Sub example — Producer/Consumer pattern
import { EventBus } from "../events/bus";
import { BaseListener } from "../events/listener";
import { Event, EventType, OrderPayload, UserPayload } from "../events/types";

// Producer: creates user and emits event
export class UserService {
  constructor(private bus: EventBus) {}

  createUser(name: string, email: string): string {
    const userId = `user_${Date.now()}`;
    this.bus.emit(EventType.USER_CREATED, { userId, name, email });
    return userId;
  }
}

// Consumer: listens for user created events
export class WelcomeEmailListener extends BaseListener<UserPayload> {
  eventType = EventType.USER_CREATED;

  async handler(event: Event<UserPayload>): Promise<void> {
    const { name, email } = event.payload;
    console.log(`[WelcomeEmail] Sending to ${email} — Welcome, ${name}!`);
    // Simulate async email send
    await new Promise((r) => setTimeout(r, 50));
  }
}

// Another consumer: analytics
export class AnalyticsListener extends BaseListener<UserPayload> {
  eventType = EventType.USER_CREATED;
  events: Event<UserPayload>[] = [];

  async handler(event: Event<UserPayload>): Promise<void> {
    this.events.push(event);
    console.log(`[Analytics] Tracked new user: ${event.payload.userId}`);
  }
}

// Order producer
export class OrderService {
  constructor(private bus: EventBus) {}

  placeOrder(
    userId: string,
    items: Array<{ sku: string; qty: number; price: number }>
  ): string {
    const orderId = `order_${Date.now()}`;
    const total = items.reduce((s, i) => s + i.qty * i.price, 0);
    this.bus.emit(EventType.ORDER_PLACED, { orderId, userId, items, total });
    return orderId;
  }
}

// Order consumer
export class OrderProcessingListener extends BaseListener<OrderPayload> {
  eventType = EventType.ORDER_PLACED;

  async handler(event: Event<OrderPayload>): Promise<void> {
    const { orderId, total } = event.payload;
    console.log(`[OrderProcessing] Processing order ${orderId} — R$${total}`);
  }
}
