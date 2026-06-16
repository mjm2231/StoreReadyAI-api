.PHONY: dev-setup up down status api db-reset db-reset-force migrate-up migrate-down migrate-version migrate-create build build-api tools gen clean-out

# ---- Config ----
# DSN precedence:
# 1) DB_DSN
# 2) storeready_ai_DB_DSN (matches internal/config env overlay)
DB_DSN ?=

MIGRATE_BIN := ./.dev/bin/migrate
MIGRATIONS_DIR := ./migrations

# ---- Build output ----
# All compiled artifacts go to repo root `./._out`
OUT_DIR := ./._out
API_BIN := $(OUT_DIR)/api

# ---- App ----
dev-setup:
	bash scripts/dev_setup.sh

up: down gen
	bash scripts/dev_start.sh

down:
	echo ">> Stopping dev environment (containers, processes, etc.)"
	bash scripts/dev_stop.sh

status:
	echo ">> Dev environment status"
	bash scripts/dev_status.sh

api:
	go run ./cmd/api

# ---- DB (dangerous) ----
# Truncate all data (keep schema). Will ask for confirmation.
db-reset:
	bash scripts/db_reset.sh

# Truncate all data (keep schema). No confirmation.
db-reset-force:
	FORCE=1 bash scripts/db_reset.sh

# ---- Build ----
build: build-api

build-api:
	@mkdir -p $(OUT_DIR)
	go build -o $(API_BIN) ./cmd/api

tools:
	@go install github.com/swaggo/swag/cmd/swag@latest

gen:
	@mkdir -p gen/swagger
	@echo ">> swag to gen/swagger"
	@swag init -g cmd/api/main.go -o gen/swagger
	@echo "http://127.0.0.1:8080/swagger/index.html"

clean-out:
	rm -rf $(OUT_DIR)


# ---- Migrations (golang-migrate) ----
# Example:
#   make migrate-up DB_DSN='root:@tcp(127.0.0.1:3306)/storeready_ai?charset=utf8mb4&parseTime=true&loc=Local'
# or:
#   export storeready_ai_DB_DSN='...'
#   make migrate-up

migrate-version:
	@$(MIGRATE_BIN) -version

migrate-up:
	@mkdir -p $(MIGRATIONS_DIR)
	@DSN="$${DB_DSN:-$${storeready_ai_DB_DSN:-}}"; \
	if [ -z "$$DSN" ]; then \
		echo "[ERR] missing DB_DSN. Provide DB_DSN or storeready_ai_DB_DSN"; \
		exit 1; \
	fi; \
	$(MIGRATE_BIN) -path $(MIGRATIONS_DIR) -database "$$DSN" up

# Roll back 1 step by default:
#   make migrate-down
# Or roll back N steps:
#   make migrate-down N=3
migrate-down:
	@mkdir -p $(MIGRATIONS_DIR)
	@DSN="$${DB_DSN:-$${storeready_ai_DB_DSN:-}}"; \
	if [ -z "$$DSN" ]; then \
		echo "[ERR] missing DB_DSN. Provide DB_DSN or storeready_ai_DB_DSN"; \
		exit 1; \
	fi; \
	N="$${N:-1}"; \
	$(MIGRATE_BIN) -path $(MIGRATIONS_DIR) -database "$$DSN" down $$N

# Create a new migration pair:
#   make migrate-create NAME=add_users
migrate-create:
	@mkdir -p $(MIGRATIONS_DIR)
	@NAME="$${NAME:-}"; \
	if [ -z "$$NAME" ]; then \
		echo "[ERR] missing NAME. Example: make migrate-create NAME=add_users"; \
		exit 1; \
	fi; \
	$(MIGRATE_BIN) create -ext sql -dir $(MIGRATIONS_DIR) -seq "$$NAME"