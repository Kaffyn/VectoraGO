// Event types and interfaces for the event-driven system

export type EventPayload = Record<string, unknown>;

export interface Event<T extends EventPayload = EventPayload> {
  id: string;
  type: string;
  payload: T;
  timestamp: Date;
  source?: string;
}

export type EventHandler<T extends EventPayload = EventPayload> = (
  event: Event<T>
) => void | Promise<void>;

export interface Subscription {
  unsubscribe(): void;
}

// Domain event types
export enum EventType {
  USER_CREATED = "user.created",
  USER_UPDATED = "user.updated",
  ORDER_PLACED = "order.placed",
  ORDER_SHIPPED = "order.shipped",
  PAYMENT_PROCESSED = "payment.processed",
  NOTIFICATION_SENT = "notification.sent",
}

// Typed payloads
export interface UserPayload extends EventPayload {
  userId: string;
  name: string;
  email: string;
}

export interface OrderPayload extends EventPayload {
  orderId: string;
  userId: string;
  items: Array<{ sku: string; qty: number; price: number }>;
  total: number;
}

export interface PaymentPayload extends EventPayload {
  paymentId: string;
  orderId: string;
  amount: number;
  method: "credit_card" | "pix" | "boleto";
}
