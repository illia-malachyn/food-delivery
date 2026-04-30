# Payment Service Business Rules (Learning Version)

This document is the source of truth for the simplified `payment` bounded context.

## Goal

The payment service records whether an order payment is successful, failed, or refunded.

## Aggregate

- Aggregate: `Payment`
- Identity: `payment_id` (UUID)
- Foreign identity: `order_id`
- Core fields: `amount`, `currency`, `status`, `failure_reason`, `created_at`

## Invariants

1. `order_id` is required.
2. `amount` must be greater than `0`.
3. `currency` must be a 3-letter uppercase ISO code.

## Lifecycle

Valid statuses:

- `pending`
- `paid`
- `failed`
- `refunded`

Allowed transitions:

1. `pending -> paid`
2. `pending -> failed`
3. `paid -> refunded`

Forbidden transitions:

1. Any transition out of `failed`.
2. Refunding `pending` or `failed`.
3. Marking `paid` or `failed` more than once.

## Event Policy

Domain events:

- `PaymentInitiated`
- `PaymentPaid`
- `PaymentFailed`
- `PaymentRefunded`

## DDD Practice Scenarios

1. Reject payment with `amount <= 0`.
2. Reject invalid status transitions.
3. Allow only `paid` payments to be refunded.
