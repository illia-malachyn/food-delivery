# Food Delivery

Educational project for learning **Event-Driven Architecture (EDA)** with **Domain-Driven Design (DDD)** in Go.

## Goal

This repository is a playground to practice:

- splitting a system into bounded-context microservices
- modeling domain logic with DDD layers (`domain`, `application`, `infrastructure`)
- evolving service-to-service communication toward event-driven flows

## Microservices

- `order` (currently has HTTP + Postgres integration)
- `payment`
- `delivery`
- `restaurant`

At this stage, `payment`, `delivery`, and `restaurant` are scaffolded services with startup entrypoints.

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
docker compose up --build
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

## Run Locally (Go)

Requirements:

- Go 1.25+

Run a service:

```bash
go run ./order/cmd/app
go run ./payment/cmd/app
go run ./delivery/cmd/app
go run ./restaurant/cmd/app
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

## Next Learning Steps

- add postgres 
- impl event publisher with outbox table solving the dual-write problem
- add message broker (kafka/rabbitmq)
