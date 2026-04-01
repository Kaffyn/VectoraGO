// Multi-step workflow via events — order placement flow
import { EventBus } from "../events/bus";
import { BaseListener } from "../events/listener";
import {
  Event,
  EventType,
  OrderPayload,
  PaymentPayload,
} from "../events/types";

type WorkflowState = "pending" | "payment_processing" | "shipped" | "done";

// ISSUE: state stored in-memory — not persistent, lost on restart
const orderStates = new Map<string, WorkflowState>();

export class PaymentListener extends BaseListener<OrderPayload> {
  eventType = EventType.ORDER_PLACED;

  constructor(private bus: EventBus) {
    super();
  }

  async handler(event: Event<OrderPayload>): Promise<void> {
    const { orderId, total } = event.payload;
    orderStates.set(orderId, "payment_processing");
    console.log(`[Payment] Processing R$${total} for order ${orderId}`);

    // Simulate payment gateway
    await new Promise((r) => setTimeout(r, 100));

    this.bus.emit(EventType.PAYMENT_PROCESSED, {
      paymentId: `pay_${Date.now()}`,
      orderId,
      amount: total,
      method: "pix",
    });
  }
}

export class ShippingListener extends BaseListener<PaymentPayload> {
  eventType = EventType.PAYMENT_PROCESSED;

  constructor(private bus: EventBus) {
    super();
  }

  async handler(event: Event<PaymentPayload>): Promise<void> {
    const { orderId } = event.payload;
    orderStates.set(orderId, "shipped");
    console.log(`[Shipping] Order ${orderId} dispatched`);

    this.bus.emit(EventType.ORDER_SHIPPED, { orderId });
    orderStates.set(orderId, "done");
  }
}

export function getOrderState(orderId: string): WorkflowState | undefined {
  return orderStates.get(orderId);
}
