# Food Delivery

Educational project for learning **Event-Driven Architecture (EDA)** with **Domain-Driven Design (DDD)** in Go.

## Goal

This repository is a playground to practice:

- splitting a system into bounded-context microservices
- modeling domain logic with DDD layers (`domain`, `application`, `infrastructure`)
- evolving service-to-service communication toward event-driven flows

## Microservices

- `order`
- `payment`
- `delivery`
- `restaurant`

## Project Structure

```text
food-delivery/
  order/
  payment/
  delivery/
  restaurant/
```

## Run With Docker Compose

Requirements:

- Docker + Docker Compose

Environment:

- configure DB settings in `.env` (example values):

```bash
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=postgres
POSTGRES_ADMIN_DB=postgres
ORDER_DB_NAME=orders
ORDER_DB_USER=orders_user
ORDER_DB_PASSWORD=orders_password
ORDER_DB_HOST=localhost
ORDER_DB_PORT=5432
ORDER_DB_SSLMODE=disable
```

Start all services:

```bash
docker compose up --build -d
```

Services started by compose:

- `postgres` on `localhost:5432`
- `order` on `localhost:9876`
- `payment`
- `delivery`
- `restaurant`

Stop:

```bash
docker compose down
```

If you change DB/user bootstrap variables after the first run, recreate the Postgres volume:

```bash
docker compose down -v
docker compose up --build
```

## Order API (Current)

Create order:

```http
POST /orders
```

Example:

```bash
curl -X POST http://localhost:9876/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u-1","item_id":"pizza","quantity":2}'
```

## Run migrations

From the `order` service directory:

```bash
cd order
```

Create migration:

```bash
make migrate-create NAME=migration_name
```

Run migrations:

```bash
make migrate-up
```

Rollback last migration:

```bash
make migrate-down
```

Show current migration version:

```bash
make migrate-version
```

## Next Learning Steps

- add postgres 
- add cloud secret manager
- host app somewhere (gcp/aws)
- impl event publisher (polling) with outbox table solving the dual-write problem
- add Debezium or self-written CDC
- add message broker (kafka/rabbitmq) with auto-commit=false
- add dedup table
- write upcaster for new versions of events
- add orchestrating saga with temporal or self-written
- add choreography saga
- add idempotent consumers
- use cqrs in some service
- use clean-arch in one service (with Presenters)
- add CI
- add CD
- add k8s
- add auth (auth or api gateway) with access and refresh tokens
- add monitoring
- add load balancer
- scale microservices horizontally
- add private VPC for a system
- ? add minVersion to cqrs commands to fix the read-your-writes problem
- build ACL in some service
- ? use event sourcing? (I want to see how Reconstitute() func work)
- 
