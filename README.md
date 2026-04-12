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

- add domain events for order lifecycle (created, paid, delivered)
- publish/consume events between services (EDA flow)
- add idempotency and retry handling for message consumers
- add tests per layer (domain first)
