# Restaurant Service Business Rules

This document is the source of truth for the `restaurant` bounded context.

## Goal

The restaurant service manages store availability, menu acceptance, kitchen capacity, and preparation lifecycle for incoming orders.

## Aggregate

- Aggregate: `KitchenOrder`
- Identity: `kitchen_order_id` (UUID)
- Foreign identity: `order_id`, `restaurant_id`
- Core fields: `item_id`, `quantity`, `status`, `accepted_at`, `ready_at`, `rejection_reason`

Supporting aggregate:

- `Restaurant` (`restaurant_id`, `is_active`, working hours, capacity profile)

## Invariants (Simple)

1. `restaurant_id` must reference an active restaurant.
2. `item_id` must exist in current published menu.
3. `quantity` must be positive and below per-item max kitchen limit.
4. Rejection reason is mandatory when status is `rejected`.
5. A single `order_id` can create at most one `KitchenOrder` per restaurant.

## Lifecycle

Valid statuses:

- `received`
- `accepted`
- `rejected`
- `preparing`
- `ready_for_pickup`

Allowed transitions:

1. `received -> accepted`
2. `received -> rejected`
3. `accepted -> preparing`
4. `preparing -> ready_for_pickup`

Forbidden transitions:

1. Moving from `rejected` to any active cooking status.
2. Returning from `ready_for_pickup` back to `preparing` without reopening flow.

## Business Policies (Harder)

1. Auto-reject timeout policy:
- If no decision within 120 seconds of `received`, system auto-rejects with reason `decision_timeout`.

2. Capacity policy:
- Kitchen has max concurrent load points.
- Each item has load weight; order accepted only if `current_load + order_load <= max_load`.

3. Dynamic prep estimate policy:
- On `accepted`, compute estimated ready time from queue depth + item prep baseline.
- If estimate exceeds SLA, reject with `sla_exceeded`.

4. Temporary 86 policy (item unavailable):
- Item can be marked unavailable for a time window.
- Existing accepted kitchen orders remain valid; new ones are rejected.

5. Change freeze policy:
- Once status is `preparing`, quantity/item changes are forbidden.
- Any customer change request must create a cancellation + new order flow.

## Event Policy

Domain events:

- `KitchenOrderReceived`
- `OrderAcceptedByRestaurant`
- `OrderRejectedByRestaurant`
- `FoodPreparationStarted`
- `FoodPrepared`

Integration events:

1. `OrderSentToRestaurant` (consumed from order context)
2. `OrderRejectedByRestaurant` (published)
3. `FoodPrepared` (published)

## Cross-Context Contract Rules

1. `FoodPrepared` must include `order_id`, `restaurant_id`, `ready_at`, `pickup_code`, `version`.
2. Restaurant must not publish `FoodPrepared` before local status is committed.
3. Duplicate `OrderSentToRestaurant` events must not create duplicate kitchen orders.

## DDD Practice Scenarios

Simple scenarios:

1. Reject order when restaurant is inactive.
2. Reject order when item is missing from menu.
3. Auto-reject when no decision in 120 seconds.

Hard scenarios:

1. Capacity race when many orders are accepted concurrently.
2. Menu version changes between receive and accept.
3. Kitchen accepted order, then receives upstream `OrderCancelled` while already `preparing`.
