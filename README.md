# Food Delivery

Educational Go project for practicing Event-Driven Architecture (EDA) and Domain-Driven Design (DDD) with multiple bounded-context services.

## Services

- `order`
- `payment`
- `delivery`
- `restaurant`

Detailed domain/event modeling notes are in [`docs/architecture.md`](docs/architecture.md).
Order-specific rules are in [`docs/order-business-rules.md`](docs/order-business-rules.md).

## Current Runtime Status

- `order`:
  - HTTP API: `POST /orders` on `localhost:9876`
  - PostgreSQL persistence
  - Outbox relay + Kafka publisher (`order.events` topic by default)
- `payment`, `delivery`, `restaurant`:
  - basic HTTP app stubs running on `:8080` inside containers
  - PostgreSQL migrations are wired
  - no host port published in `docker-compose.yml` yet

## Project Layout

```text
food-delivery/
  order/
  payment/
  delivery/
  restaurant/
  docs/
  docker-compose.yml
```

## Quick Start (Docker Compose)

Requirements:

- Docker
- Docker Compose

Root `.env` should contain DB bootstrap values (already present in this repo):

```bash
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=postgres
POSTGRES_ADMIN_DB=postgres
ORDER_DB_NAME=orders
ORDER_DB_USER=orders_user
ORDER_DB_PASSWORD=orders_password
PAYMENT_DB_NAME=payments
PAYMENT_DB_USER=payments_user
PAYMENT_DB_PASSWORD=payments_password
DELIVERY_DB_NAME=deliveries
DELIVERY_DB_USER=deliveries_user
DELIVERY_DB_PASSWORD=deliveries_password
RESTAURANT_DB_NAME=restaurants
RESTAURANT_DB_USER=restaurants_user
RESTAURANT_DB_PASSWORD=restaurants_password
```

Start everything:

```bash
docker compose up --build -d
```

Compose starts:

- `postgres` (`localhost:5432`)
- `kafka` (`localhost:9092`)
- one migration job per service (`*-migrate`)
- application containers: `order`, `payment`, `delivery`, `restaurant`

Stop:

```bash
docker compose down
```

If you change DB bootstrap variables after first run, recreate Postgres data:

```bash
docker compose down -v
docker compose up --build -d
```

## Order API

Create order:

```http
POST /orders
Content-Type: application/json
```

Example:

```bash
curl -X POST http://localhost:9876/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u-1","item_id":"pizza","quantity":2}'
```

Expected result: HTTP `200 OK` on success.

Confirm order:

```bash
curl -X POST http://localhost:9876/orders/<order-id>/confirm
```

Cancel order:

```bash
curl -X POST http://localhost:9876/orders/<order-id>/cancel \
  -H "Content-Type: application/json" \
  -d '{"reason":"payment failed"}'
```

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

- add postgres +
- use rabbitmq for service to service communication and kafka for service to services.
- use cloud secret manager
- host app somewhere (gcp/aws)
- impl event publisher (polling) with outbox table solving the dual-write problem +
- add Debezium or self-written CDC
- add message broker (kafka/rabbitmq) with auto-commit=false +
- add dedup table
- write upcaster for new versions of events +
- add orchestrating saga with temporal/cadence or self-written
- choreography saga with distributed tracing
- idempotent consumers
- use cqrs in some service
- use clean-arch's Presenters in some service
- add CI +
- add CD
- add k8s and deploy to different geo zones
- add auth service (or api gateway) (with JWT)
- monitoring/observability
- use some load balancer
- scale out microservices
- add private VPC for a system
- ? add minVersion to cqrs commands to fix the read-your-writes problem
- ? build ACL in some service
- ? use event sourcing? (I want to see how Reconstitute() func work)
- competing consumers
- performance tuning for consumers (prefetch messages, parallel processing, batch writes to db, compression, connection pool, lock timeout)