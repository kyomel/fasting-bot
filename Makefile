APP_NAME := fasting-bot
CMD := ./cmd/fasting-bot
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME)

.PHONY: help setup run build test race tidy clean

help: ## Show available commands.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Prepare local files and download dependencies.
	@if [ ! -f .env ]; then cp .env.example .env; echo "created .env from .env.example"; else echo ".env already exists"; fi
	@go mod download

run: setup ## Prepare and run the application locally.
	@go run $(CMD)

build: ## Build the application binary.
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN) $(CMD)

build-linux: ## Build for Linux deployment (cross-compile).
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(APP_NAME)-linux $(CMD)

test: ## Run package tests.
	@go test ./...

race: ## Run package tests with the race detector.
	@go test -race ./...

tidy: ## Clean up module dependencies.
	@go mod tidy

clean: ## Remove build output and local SQLite files.
	@rm -rf $(BIN_DIR)
	@rm -f *.db *.db-*
