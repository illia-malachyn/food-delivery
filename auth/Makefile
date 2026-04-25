.PHONY: migrate-up migrate-down migrate-create migrate-force migrate-version regen-mocks test test-integration

# Load environment variables from .env file
include .env
export

# Database connection string
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# Migrations directory
MIGRATIONS_DIR := database/postgres/migrations

# Apply all pending migrations
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

# Rollback the last migration
migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

# Create a new migration file (usage: make migrate-create NAME=migration_name)
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME) -digits 3

# Show current migration version
migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

regen-mocks:
	@dirs=$$(rg -l 'type[[:space:]]+[A-Za-z0-9_]+[[:space:]]+interface[[:space:]]*\{' -g'*.go' | while read -r file; do dirname "$$file"; done | sort -u); \
	if [ -z "$$dirs" ]; then \
		echo "No interfaces found"; \
		exit 0; \
	fi; \
	for dir in $$dirs; do \
		echo "==> mockery: $$dir"; \
		go run github.com/vektra/mockery/v2@v2.53.5 --all --dir "$$dir" --output "$$dir/mocks" --outpkg mocks --with-expecter; \
	done

test:
	go test ./...

test-integration:
	go test -tags=integration ./integration_tests -v
