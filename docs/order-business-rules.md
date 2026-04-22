# Order Service Business Rules

This document is the source of truth for the `order` bounded context.

## Goal

The order service accepts customer intent, validates business invariants, and emits reliable integration events for downstream services (payment, restaurant, delivery).

## Aggregate

- Aggregate: `Order`
- Identity: `order_id` (string)
- Core fields: `user_id`, `item_id`, `quantity`, `status`

## Invariants

1. `user_id` must be non-blank.
2. `item_id` must be non-blank.
3. `quantity` must be between `1` and `50` inclusive.
4. Cancellation reason must be meaningful: at least 5 non-space characters.
5. IDs and cancellation reason are normalized by trimming spaces.

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

Rationale for cancellation policy:

- Once an order is confirmed, operational flow is considered committed. Late cancellation should be handled by a different business flow (e.g. support/refund), not by a simple order state change.

## Event Policy

Domain events:

- `OrderPlaced`
- `OrderConfirmed`
- `OrderCancelled`

Integration events emitted to other contexts:

1. `OrderPlaced` (published)
2. `OrderCancelled` (published)

Internal-only domain event:

1. `OrderConfirmed` (not published cross-service)

Rationale:

- `OrderConfirmed` is a local state transition and does not provide a stable contract for external services in current architecture.
