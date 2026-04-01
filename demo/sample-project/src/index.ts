import { EventBus } from "./events/bus";
import {
  UserService,
  WelcomeEmailListener,
  AnalyticsListener,
  OrderService,
  OrderProcessingListener,
} from "./examples/pubsub";
import { PaymentListener, ShippingListener, getOrderState } from "./examples/workflow";

async function main() {
  const bus = new EventBus();

  // Wire up listeners
  const welcome = new WelcomeEmailListener();
  const analytics = new AnalyticsListener();
  const orderProc = new OrderProcessingListener();
  const payment = new PaymentListener(bus);
  const shipping = new ShippingListener(bus);

  welcome.listen(bus);
  analytics.listen(bus);
  orderProc.listen(bus);
  payment.listen(bus);
  shipping.listen(bus);

  // Run pub/sub demo
  const userSvc = new UserService(bus);
  const userId = userSvc.createUser("Ana", "ana@example.com");

  const orderSvc = new OrderService(bus);
  const orderId = orderSvc.placeOrder(userId, [
    { sku: "BOOK-001", qty: 2, price: 49.9 },
    { sku: "PEN-002", qty: 5, price: 3.5 },
  ]);

  // Give async handlers time to complete
  await new Promise((r) => setTimeout(r, 500));

  console.log(`\nOrder state: ${getOrderState(orderId)}`);
  console.log(`Events in history: ${bus.getHistory().length}`);
}

main().catch(console.error);
