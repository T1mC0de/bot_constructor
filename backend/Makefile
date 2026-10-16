# Load local settings from .env if it exists and pass them to the commands.
-include .env
export

.PHONY: run run-core run-bot build test lint

# Start core and the bot together. Ctrl+C stops both.
run: build
	@bin/core & core=$$!; trap 'kill $$core 2>/dev/null' EXIT; bin/bot

run-core: ## Only the core (HTTP API + storage)
	go run ./core/cmd/core

run-bot: ## Only the bot
	go run ./bot/cmd/bot

build:
	go build -o bin/core ./core/cmd/core
	go build -o bin/bot ./bot/cmd/bot

test:
	go test ./... -race

lint:
	go vet ./...
