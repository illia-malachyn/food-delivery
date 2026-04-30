# Food Delivery

Educational Go project for practicing Event-Driven Architecture (EDA) and Domain-Driven Design (DDD) with multiple bounded-context services.

## Services

- `order`
- `payment`
- `delivery`
- `restaurant`
- `auth`

Detailed domain/event modeling notes are in [`docs/architecture.md`](docs/architecture.md).
Business rules by service:

- [`docs/order-business-rules.md`](docs/order-business-rules.md)
- [`docs/payment-business-rules.md`](docs/payment-business-rules.md)
- [`docs/restaurant-business-rules.md`](docs/restaurant-business-rules.md)
- [`docs/delivery-business-rules.md`](docs/delivery-business-rules.md)

## Current Runtime Status

- `gateway`:
  - single host entrypoint on `localhost:8080`
  - routes `/auth/*` -> `auth:8080`
  - routes `/orders*` -> `order:8080`
- `prometheus`:
  - metrics UI and query API on `localhost:9090`
  - scrapes `auth`, `order`, `payment`, `delivery`, `restaurant` on `/metrics`
- `grafana`:
  - dashboards UI on `localhost:3000` (default creds: `admin` / `admin`)
  - provisioned Prometheus datasource + preloaded dashboard
- `order`:
  - HTTP API served behind gateway (`localhost:8080/orders`)
  - PostgreSQL persistence
  - Outbox relay + Kafka publisher (`order.events` topic by default)
- `payment`, `delivery`, `restaurant`:
  - basic HTTP app stubs running on `:8080` inside containers
  - PostgreSQL migrations are wired
  - no host port published in `docker-compose.yml` yet
- `auth`:
  - HTTP API served behind gateway (`localhost:8080/auth/*`)
  - PostgreSQL users table + Redis-backed refresh-token sessions
  - JWT access token in JSON response + refresh token in HttpOnly cookie
  - middleware chain infra (`recovery`, `logging`, `tracing`, `metrics`, `auth`)

## Project Layout

```text
food-delivery/
  order/
  payment/
  delivery/
  restaurant/
  auth/
  docs/
  docker-compose.yml
```

## Quick Start (Docker Compose)

Requirements:

- Docker
- Docker Compose

Create root `.env` from the template:

```bash
make env-init
```

Start everything:

```bash
docker compose up --build -d
```

Compose starts:

- `gateway` (`localhost:8080`)
- `prometheus` (`localhost:9090`)
- `grafana` (`localhost:3000`)
- `postgres` (`localhost:5432`)
- `redis` (`localhost:6379`)
- `kafka` (`localhost:9092`)
- one migration job per service (`*-migrate`)
- application containers: `order`, `payment`, `delivery`, `restaurant`, `auth`

Stop:

```bash
docker compose down
```

If you change DB bootstrap variables after first run, recreate Postgres data:

```bash
docker compose down -v
docker compose up --build -d
```

## OpenAPI Specs

Visual API docs are published with Redoc via GitHub Pages:

- site: <https://illia-malachyn.github.io/food-delivery/>
- per service:
  - <https://illia-malachyn.github.io/food-delivery/api/auth.html>
  - <https://illia-malachyn.github.io/food-delivery/api/order.html>
  - <https://illia-malachyn.github.io/food-delivery/api/payment.html>
  - <https://illia-malachyn.github.io/food-delivery/api/delivery.html>
  - <https://illia-malachyn.github.io/food-delivery/api/restaurant.html>

Quick Swagger UI preview for any spec:

```bash
docker run --rm -p 8088:8080 \
  -e SWAGGER_JSON=/spec/openapi.yaml \
  -v "$(pwd)/auth:/spec" \
  swaggerapi/swagger-ui
```

Use `-v "$(pwd)/order:/spec"` (or another service directory) to preview a different API spec.

Validate all specs:

```bash
make openapi-lint
```

## End-to-End Tests

The repository has a deliberately small E2E suite for critical user/business flows only.

Current E2E coverage:

- Auth via gateway: register -> login -> me -> refresh -> logout
- Order via gateway + DB assertions: create -> outbox OrderPlaced(v2) -> cancel -> outbox OrderCancelled

Run:

```bash
make e2e-tests
```

Details and environment overrides are documented in [`tests/e2e/README.md`](tests/e2e/README.md).

## Metrics

Each microservice exposes Prometheus metrics on `/metrics`:

- `auth:8080/metrics`
- `order:8080/metrics`
- `payment:8080/metrics`
- `delivery:8080/metrics`
- `restaurant:8080/metrics`

Prometheus is configured via [`observability/prometheus.yml`](observability/prometheus.yml) and available on `http://localhost:9090`.
Custom endpoint request counter in every service:

- `http_requests_total{service,method,path,status}`

Grafana provisioning files:

- datasource: [`observability/grafana/provisioning/datasources/datasource.yml`](observability/grafana/provisioning/datasources/datasource.yml)
- dashboard provider: [`observability/grafana/provisioning/dashboards/dashboards.yml`](observability/grafana/provisioning/dashboards/dashboards.yml)
- dashboard JSON: [`observability/grafana/dashboards/food-delivery-overview.json`](observability/grafana/dashboards/food-delivery-overview.json)

Grafana access:

- URL: `http://localhost:3000`
- credentials: `admin` / `admin`
- preloaded dashboard: `Food Delivery Overview`

## Migrations

The project has:

- root `Makefile` to run migration commands across services
- per-service `Makefile` in each service directory

Root usage:

```bash
# all services
make migrate-up-all
make migrate-down-all
make migrate-version-all
make migrate-create-all NAME=add_new_column

# single service
make migrate-up SERVICE=order
make migrate-down SERVICE=payment
make migrate-version SERVICE=delivery
make migrate-create SERVICE=restaurant NAME=add_index
```

Per-service usage (example: `order/`):

```bash
cd order
make migrate-up
make migrate-down
make migrate-version
make migrate-create NAME=add_new_column
```

Note: these targets require the `migrate` CLI in your PATH.

## Next Learning Steps

Legend: `+` done, `~` partially done, no marker = todo. `?` = considering.

### Foundations (done)

- `+` PostgreSQL per service (logically separated DBs on shared instance)
- `+` Auth service (JWT access + refresh, Redis sessions, middleware chain)
- `+` API gateway (nginx path-routing)
- `+` CI (GitHub Actions: tests + docker build)
- `+` Observability baseline (Prometheus metrics, Grafana dashboards)

### Eventing — producer side

- `+` Outbox pattern with polling relay (solves dual-write)
- `+` Kafka publisher with manual commits (`auto-commit=false`)
- `~` Event versioning + upcasters (done for `OrderPlaced` only — generalize to all events)
- CDC-based outbox publishing with Debezium (alternative to polling relay; compare trade-offs)

### Eventing — consumer side (next focus)

- Kafka consumer skeleton with manual commits, consumer groups, graceful shutdown
- Idempotent consumers via dedup table (`message_id` PK, processed_at) — prerequisite for everything below
- Dead-letter queue + retry policy with exponential backoff and jitter
- Poison-message handling (parse failures, schema mismatches)
- Competing consumers / consumer scaling (partition count, consumer group sizing)
- Consumer performance tuning: prefetch / batch fetch, parallel processing per partition, batch DB writes, compression, connection pool sizing, lock timeouts

### Cross-service workflows

- Orchestration saga (Temporal or self-written state machine) — e.g. order → payment → restaurant → delivery with compensations
- Choreography saga with distributed tracing (OpenTelemetry) — same flow, no central orchestrator
- Decide RabbitMQ vs Kafka per use case: RPC-style commands (RabbitMQ) vs domain events (Kafka), and document the reasoning

### Resilience

- Circuit breaker on outbound HTTP calls (e.g. payment provider stub)
- Retries with exponential backoff + jitter (consumer side and HTTP clients)
- Timeouts and bulkheads on shared resources (DB pools, HTTP clients)

### Architectural patterns (pick one or two services)

- CQRS with separate read model (likely `restaurant` — heavy read, light write)
- Clean-architecture Presenters in one service (compare to current handler-returns-DTO style)
- `?` Event sourcing in one aggregate to see `Reconstitute()` in action
- `?` Anti-corruption layer in a service that integrates with an external system
- `?` `minVersion` on CQRS commands to fix read-your-writes consistency

### Shared platform code

- Extract `shared/` module: middleware chain, outbox relay, Kafka consumer base, integration-event interface, upcaster pattern
- Replace counter-based domain ID generator with UUID/ULID (current impl breaks with >1 replica)

### Deployment & infrastructure

- Cloud secret manager (AWS Secrets Manager / SSM Parameter Store)
- Host on AWS (start: single EC2 + docker-compose; target: ECS + RDS + SNS/SQS or MSK)
- Private VPC with public/private subnet split
- Load balancer (ALB) replacing nginx gateway, with TLS via ACM/Let's Encrypt
- CD pipeline (GitHub Actions → ECR → ECS deploy)
- Kubernetes deployment, eventually multi-region
- Scale out with KEDA Scaler

### Stretch

- Scale out one service horizontally and observe partition rebalancing, sticky sessions, and cache invalidation behavior
