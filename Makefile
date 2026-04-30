.PHONY: env-init migrate-up-all migrate-down-all migrate-version-all migrate-create-all migrate-up migrate-down migrate-version migrate-create regen-mocks openapi-lint unit-tests integration-tests e2e-tests

SERVICES := order payment delivery restaurant auth
SERVICE ?=
NAME ?=
OPENAPI_SPECS := auth/openapi.yaml order/openapi.yaml payment/openapi.yaml delivery/openapi.yaml restaurant/openapi.yaml

env-init:
	@for pair in \
		".env.example:.env" \
		"order/.env.example:order/.env" \
		"payment/.env.example:payment/.env" \
		"delivery/.env.example:delivery/.env" \
		"restaurant/.env.example:restaurant/.env" \
		"auth/.env.example:auth/.env"; do \
		src="$${pair%%:*}"; \
		dst="$${pair##*:}"; \
		if [ ! -f "$$src" ]; then \
			echo "Missing template: $$src"; \
			exit 1; \
		fi; \
		if [ -f "$$dst" ]; then \
			echo "exists: $$dst"; \
		else \
			cp "$$src" "$$dst"; \
			echo "created: $$dst"; \
		fi; \
	done

migrate-up-all:
	@for svc in $(SERVICES); do \
		echo "==> $$svc: migrate-up"; \
		$(MAKE) -C $$svc migrate-up; \
	done

migrate-down-all:
	@for svc in $(SERVICES); do \
		echo "==> $$svc: migrate-down"; \
		$(MAKE) -C $$svc migrate-down; \
	done

migrate-version-all:
	@for svc in $(SERVICES); do \
		echo "==> $$svc: migrate-version"; \
		$(MAKE) -C $$svc migrate-version; \
	done

migrate-create-all:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create-all NAME=migration_name"; \
		exit 1; \
	fi
	@for svc in $(SERVICES); do \
		echo "==> $$svc: migrate-create NAME=$(NAME)"; \
		$(MAKE) -C $$svc migrate-create NAME=$(NAME); \
	done

migrate-up migrate-down migrate-version migrate-create:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Error: SERVICE is required. Usage: make $@ SERVICE=order"; \
		exit 1; \
	fi
	$(MAKE) -C $(SERVICE) $@ NAME=$(NAME)

regen-mocks:
	@for svc in $(SERVICES); do \
		echo "==> $$svc: regen-mocks"; \
		$(MAKE) -C $$svc regen-mocks; \
	done

openapi-lint:
	docker run --rm -v "$(PWD):/work" -w /work redocly/cli lint $(OPENAPI_SPECS)

unit-tests:
	@if [ -n "$(SERVICE)" ]; then \
		echo "==> $(SERVICE): unit-tests"; \
		(cd $(SERVICE) && GOWORK=off go test ./...); \
	else \
		for svc in $(SERVICES); do \
			echo "==> $$svc: unit-tests"; \
			(cd $$svc && GOWORK=off go test ./...); \
		done; \
	fi

e2e-tests:
	go test -tags=e2e ./tests/e2e -v

integration-tests:
	@if [ -n "$(SERVICE)" ]; then \
		if [ -d "$(SERVICE)/integration_tests" ]; then \
			echo "==> $(SERVICE): integration-tests"; \
			(cd $(SERVICE) && GOWORK=off go test -tags=integration ./integration_tests/... -v); \
		else \
			echo "==> $(SERVICE): no integration_tests/ folder, skipping"; \
		fi; \
	else \
		for svc in $(SERVICES); do \
			if [ -d "$$svc/integration_tests" ]; then \
				echo "==> $$svc: integration-tests"; \
				(cd $$svc && GOWORK=off go test -tags=integration ./integration_tests/... -v); \
			else \
				echo "==> $$svc: no integration_tests/ folder, skipping"; \
			fi; \
		done; \
	fi
