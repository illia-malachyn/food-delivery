# Order Service Business Rules

This document is the source of truth for the `order` bounded context.

## Goal

The order service accepts customer intent, validates business invariants, and emits reliable integration events for downstream services (`payment`, `restaurant`, `delivery`).

## Aggregate

- Aggregate: `Order`
- Identity: `order_id` (string)
- Core fields: `user_id`, `item_id`, `quantity`, `status`, `placed_at`, `confirmed_at`, `cancelled_at`, `cancellation_reason`

## Invariants (Simple)

1. `user_id` must be non-blank.
2. `item_id` must be non-blank.
3. `quantity` must be between `1` and `50` inclusive.
4. Cancellation reason must contain at least 5 non-space characters.
5. IDs and cancellation reason are normalized by trimming spaces.
6. An order can only represent a single menu item (multi-item carts are out of scope for this context).

## Lifecycle

Valid statuses:

- `draft`
- `placed`
- `confirmed`
- `cancelled`

Allowed transitions:

1. `draft -> placed` via `Place`.
2. `placed -> confirmed` via `Confirm`.
3. `placed -> cancelled` via `Cancel`.

Forbidden transitions:

1. Placing non-draft orders.
2. Confirming non-placed orders.
3. Cancelling draft/confirmed/cancelled orders.

## Business Policies (Harder)

1. Place timeout policy:
- If an order stays in `draft` for more than 15 minutes, it expires and cannot be placed.
- Expiration emits `OrderExpired` (domain event, internal by default).

2. Confirmation window policy:
- `payment` and `restaurant` decisions must arrive within 10 minutes of `OrderPlaced`.
- If either fails or times out, order transitions to `cancelled` with machine reason (`payment_timeout`, `restaurant_rejected`, etc.).

3. Idempotent command policy:
- `Place`, `Confirm`, and `Cancel` must accept an idempotency key.
- Repeated command with same key and same payload returns previous result.
- Repeated command with same key but different payload is rejected.

4. Ownership policy:
- Only the same `user_id` who created the order may cancel it through self-service.
- Operator cancellation is allowed only with explicit reason code and operator ID.

5. Compensation policy:
- If order is cancelled after payment capture was requested, emit `OrderCancellationRequested` so payment context can refund/void.
- `order` never directly changes payment state; it only emits intent.

## Event Policy

Domain events:

- `OrderPlaced`
- `OrderConfirmed`
- `OrderCancelled`
- `OrderExpired` (optional advanced scenario)

Integration events emitted to other contexts:

1. `OrderPlaced` (published)
2. `OrderCancelled` (published)
3. `OrderConfirmed` (optional once delivery pipeline is wired)

Internal-only domain events:

1. `OrderExpired` (may stay internal until timeout handling is externalized)

## Cross-Context Contract Rules

1. Integration event schema is versioned (`version` field required).
2. Event identity is `(event_name, aggregate_id, version, occurred_at)`.
3. Ordering is guaranteed only per `order_id`, not globally.
4. Consumers must treat duplicate events as normal and handle them idempotently.

## DDD Practice Scenarios

Simple scenarios:

1. Reject placing an order with blank `item_id`.
2. Reject cancelling with reason `"bad"` (length < 5 after trim).
3. Reject `Confirm` when status is `draft`.

Hard scenarios:

1. Two concurrent commands: `Cancel` and `Confirm` on same `placed` order (optimistic locking conflict + deterministic winner rule).
2. `OrderPlaced(v1)` and `OrderPlaced(v2)` coexist (upcaster in consumer path).
3. Timeout-driven cancellation races with late `PaymentConfirmed` event.
