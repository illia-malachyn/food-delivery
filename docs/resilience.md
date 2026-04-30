# Resilience

This project now has a small shared resilience layer used by every service runtime path.

## Every Service

All services use shared resilience/configuration for:

- HTTP request deadlines through `resilience.NewTimeoutHandler`
- HTTP server read, write, and idle timeouts
- PostgreSQL pool limits, connection lifetime, idle lifetime, health checks, and connect timeout

Per-service overrides can prefix the shared keys, for example `ORDER_DB_MAX_CONNS`, `PAYMENT_DB_CONNECT_TIMEOUT`, or `AUTH_DB_HEALTH_CHECK_PERIOD`.

## Payment Outbound Calls

- `payment` calls a payment provider through `payment/infrastructure/provider.HTTPPaymentProvider`.
- Provider calls are wrapped with an HTTP bulkhead, retries, and a circuit breaker.
- Retryable HTTP provider failures are `429` and `5xx`; network errors also count as failures.
- When the circuit is open, calls fail fast with `resilience.ErrCircuitOpen` instead of tying up request or consumer work.

Payment provider knobs:

```text
PAYMENT_PROVIDER_URL=http://payment-provider:8080
PAYMENT_PROVIDER_CIRCUIT_FAILURE_THRESHOLD=5
PAYMENT_PROVIDER_CIRCUIT_OPEN_TIMEOUT=30s
```

## Retries

- Outbound HTTP clients use bounded retries with exponential backoff and jitter.
- The payment Kafka consumer retries `HandleMessage` before logging the failure and committing the message.
- Retry settings are intentionally small by default so the system does not hide persistent downstream failures.

Payment provider HTTP retry knobs:

```text
PAYMENT_PROVIDER_HTTP_RETRY_MAX_ATTEMPTS=3
PAYMENT_PROVIDER_HTTP_RETRY_INITIAL_BACKOFF=100ms
PAYMENT_PROVIDER_HTTP_RETRY_MAX_BACKOFF=2s
PAYMENT_PROVIDER_HTTP_RETRY_JITTER=0.2
```

Payment consumer retry knobs:

```text
PAYMENT_CONSUMER_RETRY_MAX_ATTEMPTS=3
PAYMENT_CONSUMER_RETRY_INITIAL_BACKOFF=100ms
PAYMENT_CONSUMER_RETRY_MAX_BACKOFF=2s
PAYMENT_CONSUMER_RETRY_JITTER=0.2
```

## Timeouts And Bulkheads

- All HTTP handlers are wrapped with a request timeout.
- All HTTP servers configure read, write, and idle timeouts.
- Services with PostgreSQL access use pgx pool limits and connect timeouts.
- Outbound payment-provider HTTP calls use a max-concurrency bulkhead.

Shared PostgreSQL knobs:

```text
DB_MAX_CONNS=10
DB_MIN_CONNS=1
DB_CONNECT_TIMEOUT=5s
DB_MAX_CONN_LIFETIME=30m
DB_MAX_CONN_IDLE_TIME=5m
DB_HEALTH_CHECK_PERIOD=30s
```

Service-specific overrides can prefix the same keys, for example `ORDER_DB_MAX_CONNS` or `PAYMENT_DB_CONNECT_TIMEOUT`.

HTTP timeout knobs:

```text
HTTP_REQUEST_TIMEOUT=3s
HTTP_READ_TIMEOUT=10s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
```

Payment provider bulkhead knob:

```text
PAYMENT_PROVIDER_HTTP_MAX_CONCURRENT=10
```

## Local Payment Provider Stub

Docker Compose runs `payment-provider` from the payment image. The stub exposes `POST /capture`, `POST /refund`, and `GET /health` so the `payment` service has a real outbound HTTP target in local and CI stacks.

Stub knobs:

```text
PROVIDER_STUB_PORT=8080
PROVIDER_STUB_STATUS=204
PROVIDER_STUB_DELAY=0s
```

Use `PROVIDER_STUB_STATUS` and `PROVIDER_STUB_DELAY` to exercise retry, timeout, bulkhead, and circuit-breaker behavior locally.
