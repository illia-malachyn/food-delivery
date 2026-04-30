# E2E Suite

This suite is intentionally small and only covers critical cross-service flows.

Current flows:

1. `auth_service_test.go` -> `TestAuthGatewayFlow`
   - `register -> login -> me -> refresh -> logout -> refresh(401)`
2. `order_service_test.go` -> `TestOrderGatewayFlowWithOutboxAssertions`
   - `create order -> assert orders row + outbox OrderPlaced(v2) -> cancel -> assert cancelled row + outbox OrderCancelled`

Shared setup/utilities live in `helpers_test.go`.

## Why small

For this project, keep E2E tests in the `5-15` scenario range and push most coverage to unit/service-acceptance tests.

## Run

Start dependencies and services first:

```bash
make env-init
make jwt-keys
docker compose up --build -d
```

Run E2E suite:

```bash
make e2e-tests
```

Optional environment overrides:

- `E2E_BASE_URL` (default: `http://localhost:8080`)
- `E2E_ORDER_DB_DSN` (default: `postgres://orders_user:orders_password@localhost:5432/orders?sslmode=disable`)
