# Payment Service Business Rules

This document is the source of truth for the `payment` bounded context.

## Goal

The payment service authorizes/captures funds for orders, handles retries and refunds, and publishes payment outcomes reliably.

## Aggregate

- Aggregate: `Payment`
- Identity: `payment_id` (UUID)
- Foreign identity: `order_id`
- Core fields: `amount`, `currency`, `status`, `provider_transaction_id`, `failure_reason`, `created_at`

## Invariants (Simple)

1. `order_id` is required.
2. `amount` must be greater than `0`.
3. `currency` must be a 3-letter uppercase ISO code.
4. `provider_transaction_id` must be unique per provider.
5. One active payment attempt per order at a time.

## Lifecycle

Valid statuses:

- `pending`
- `authorized`
- `captured`
- `failed`
- `refunded`
- `voided`

Allowed transitions:

1. `pending -> authorized`
2. `authorized -> captured`
3. `authorized -> voided`
4. `pending -> failed`
5. `captured -> refunded` (full or partial)

Forbidden transitions:

1. Capturing `pending` directly (unless provider is immediate-capture and mapped internally to authorize+capture).
2. Refunding `authorized` (must void instead).
3. Any transition out of `failed` except creating a new payment attempt.

## Business Policies (Harder)

1. Attempt policy:
- Max 3 payment attempts per order in 15 minutes.
- The 4th attempt is blocked with `PaymentLocked` reason `too_many_attempts`.

2. Exactly-once capture intent:
- For a given `order_id`, only one capture may succeed.
- Duplicate capture requests must return already-captured response.

3. Asynchronous provider policy:
- Provider webhook may arrive before synchronous API response.
- State machine must accept out-of-order callbacks idempotently.

4. Partial refund policy:
- Total refunded amount cannot exceed captured amount.
- Each partial refund must include business reason code.

5. Fraud-hold policy:
- High-risk score payments move to `pending_review` (substate under `pending`).
- Orders in fraud hold cannot be confirmed until manual decision.

## Event Policy

Domain events:

- `PaymentInitiated`
- `PaymentAuthorized`
- `PaymentCaptureRequested`
- `PaymentCaptured`
- `PaymentFailed`
- `PaymentRefunded`
- `PaymentVoided`

Integration events:

1. `PaymentConfirmed` (maps from `PaymentCaptured`)
2. `PaymentFailed`
3. `PaymentRefunded`

## Cross-Context Contract Rules

1. `PaymentConfirmed` must include `order_id`, `payment_id`, `amount`, `currency`, `occurred_at`, `version`.
2. Payment service must publish only after local transaction commits (outbox rule).
3. Consumers must deduplicate by provider callback ID and integration event ID.

## DDD Practice Scenarios

Simple scenarios:

1. Reject payment with `amount <= 0`.
2. Reject refund greater than captured total.
3. Ignore duplicate successful webhook.

Hard scenarios:

1. Callback race: `PaymentFailed` arrives, then delayed `PaymentCaptured` arrives from provider.
2. Retry policy with circuit breaker for provider outage.
3. Refund saga after delivery failure with partial compensation fees.
