.PHONY: migrate-up-all migrate-down-all migrate-version-all migrate-create-all migrate-up migrate-down migrate-version migrate-create

SERVICES := order payment delivery restaurant
SERVICE ?=
NAME ?=

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
