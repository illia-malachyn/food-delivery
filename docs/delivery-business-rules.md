# Delivery Service Business Rules

This document is the source of truth for the `delivery` bounded context.

## Goal

The delivery service assigns couriers, tracks pickup/dropoff lifecycle, and reports final delivery outcomes.

## Aggregate

- Aggregate: `Delivery`
- Identity: `delivery_id` (UUID)
- Foreign identity: `order_id`
- Core fields: `courier_id`, `status`, `pickup_address`, `delivery_address`, `estimated_delivery_at`, `delivered_at`, `failure_reason`

## Invariants (Simple)

1. `order_id` is required.
2. `pickup_address` and `delivery_address` must be present before assignment.
3. `courier_id` is required for statuses after `courier_assigned`.
4. `delivered_at` is required only when status is `delivered`.
5. Only one active delivery per order.

## Lifecycle

Valid statuses:

- `created`
- `courier_assigned`
- `picked_up`
- `in_transit`
- `delivered`
- `failed`
- `cancelled`

Allowed transitions:

1. `created -> courier_assigned`
2. `courier_assigned -> picked_up`
3. `picked_up -> in_transit`
4. `in_transit -> delivered`
5. `courier_assigned|picked_up|in_transit -> failed`
6. `created|courier_assigned -> cancelled`

Forbidden transitions:

1. `created -> delivered` (skip-proof of movement).
2. Any transition out of `delivered`.
3. Cancelling after `picked_up` without explicit exception flow.

## Business Policies (Harder)

1. Assignment SLA policy:
- Courier must be assigned within 5 minutes after `FoodPrepared`.
- If SLA fails, emit `DeliveryAssignmentDelayed` and escalate dispatch.

2. Courier suitability policy:
- Courier can be assigned only if online, within service radius, and capacity allows one more active order.

3. Reassignment policy:
- If courier does not pick up in 8 minutes, delivery may be reassigned once.
- After max reassignments, mark `failed` with reason `pickup_timeout`.

4. Proof-of-delivery policy:
- Transition to `delivered` requires proof type (`pin` or `photo`) and geolocation accuracy <= 100m.

5. Failure compensation policy:
- If status becomes `failed` after pickup, emit `DeliveryFailed` with compensation hint (`full_refund`, `partial_refund`, `redelivery`).

## Event Policy

Domain events:

- `CourierAssigned`
- `OrderPickedUp`
- `DeliveryInTransit`
- `OrderDelivered`
- `DeliveryFailed`

Integration events:

1. `OrderDelivered`
2. `DeliveryFailed`
3. `DeliveryAssignmentDelayed` (optional advanced)

## Cross-Context Contract Rules

1. `OrderDelivered` must include `order_id`, `delivery_id`, `courier_id`, `delivered_at`, `version`.
2. Delivery service consumes `FoodPrepared` idempotently.
3. Delivery events are immutable facts; correction is done by compensating event, not update.

## DDD Practice Scenarios

Simple scenarios:

1. Reject delivery creation when addresses are missing.
2. Reject `delivered` without proof data.
3. Prevent second active delivery for same order.

Hard scenarios:

1. Courier app reports `picked_up` twice with network retries.
2. Courier marked offline during `in_transit`.
3. Order cancelled right after pickup event was already emitted.
