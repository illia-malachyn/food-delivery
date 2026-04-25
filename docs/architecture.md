# Architecture

## Business Rules Documents

- [`order` rules](./order-business-rules.md)
- [`payment` rules](./payment-business-rules.md)
- [`restaurant` rules](./restaurant-business-rules.md)
- [`delivery` rules](./delivery-business-rules.md)

## Business Rules (Event Storming)

### Event Types

- `Domain` events: business state changes produced and consumed inside the same bounded context.
- `Integration` events: cross-context contracts published for other services.

### 1. Order Context (`order` service)

- `OrderPlaced` - `Integration`
- `OrderConfirmed` - `Domain`
- `OrderCancelled` - `Integration`

### 2. Payment Context (`payment` service)

- `PaymentInitiated` - `Domain`
- `PaymentConfirmed` - `Integration`
- `PaymentRefunded` - `Integration`

### 3. Restaurant Context (`restaurant` service)

- `OrderSentToRestaurant` - `Integration`
- `OrderAcceptedByRestaurant` - `Domain`
- `OrderRejectedByRestaurant` - `Integration`
- `FoodPrepared` - `Integration`

### 4. Delivery Context (`delivery` service)

- `CourierAssigned` - `Domain`
- `OrderPickedUp` - `Domain`
- `OrderDelivered` - `Integration`
- `DeliveryFailed` - `Integration`

## System Context Map

```mermaid
flowchart LR
    Customer[Customer]
    Order[Order Service]
    Payment[Payment Service]
    Restaurant[Restaurant Service]
    Delivery[Delivery Service]
    Kafka[(Kafka / Event Bus)]

    Customer -->|Create/Cancel Order| Order
    Order -->|Publish: OrderPlaced / OrderCancelled| Kafka
    Kafka --> Payment
    Kafka --> Restaurant
    Restaurant -->|Publish: FoodPrepared / OrderRejectedByRestaurant| Kafka
    Kafka --> Delivery
    Delivery -->|Publish: OrderDelivered / DeliveryFailed| Kafka
    Payment -->|Publish: PaymentConfirmed / PaymentFailed / PaymentRefunded| Kafka
    Kafka --> Order
```

## Workflow: Happy Path

```mermaid
sequenceDiagram
    autonumber
    participant C as Customer
    participant O as Order
    participant P as Payment
    participant R as Restaurant
    participant D as Delivery

    C->>O: PlaceOrder(user_id, item_id, quantity)
    O-->>P: OrderPlaced
    O-->>R: OrderPlaced / OrderSentToRestaurant
    P-->>O: PaymentConfirmed
    R-->>O: OrderAcceptedByRestaurant
    R-->>D: FoodPrepared
    D-->>O: OrderDelivered
    O-->>C: Order completed
```

## Workflow: Restaurant Rejects Order

```mermaid
sequenceDiagram
    autonumber
    participant C as Customer
    participant O as Order
    participant P as Payment
    participant R as Restaurant

    C->>O: PlaceOrder(...)
    O-->>P: OrderPlaced
    O-->>R: OrderPlaced / OrderSentToRestaurant
    R-->>O: OrderRejectedByRestaurant(reason)
    O->>O: Cancel order (machine reason)
    O-->>P: OrderCancelled
    P-->>O: PaymentRefunded or PaymentVoided
    O-->>C: Rejected + compensation result
```

## Workflow: Delivery Fails After Pickup

```mermaid
sequenceDiagram
    autonumber
    participant O as Order
    participant P as Payment
    participant R as Restaurant
    participant D as Delivery
    participant C as Customer

    R-->>D: FoodPrepared
    D-->>O: CourierAssigned
    D-->>O: OrderPickedUp
    D-->>O: DeliveryFailed(reason, compensation_hint)
    O-->>P: OrderCancellationRequested
    P-->>O: PaymentRefunded(partial/full)
    O-->>C: Notify failure + refund outcome
```
