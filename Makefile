include .env
export

.DEFAULT_GOAL := help
SHELL := /bin/sh
COMPOSE := docker compose
MIGRATE := $(COMPOSE) run --rm todoapp-postgres-migrate -path /migrations -database 'postgres://todoapp-postgres:5432/?sslmode=disable&search_path=public'

.PHONY: help env-up env-down env-cleanup env-port-forward env-port-close migrate-create migrate-up migrate-down

help: ## Показать команды
	@awk 'BEGIN {FS = ":.*## "} /^[a-z-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' Makefile

env-up: out/go.mod ## Запустить PostgreSQL и дождаться готовности
	@$(COMPOSE) up -d --wait todoapp-postgres

env-down: ## Остановить окружение, сохранив данные
	@$(COMPOSE) --profile tools down

env-cleanup: ## Удалить данные PostgreSQL после подтверждения
	@printf 'Очистить out/pgdata? Все данные будут потеряны. [y/N]: '; \
	read -r ans; \
	if [ "$$ans" = "y" ]; then \
		$(COMPOSE) --profile tools down && \
		$(COMPOSE) run --rm --no-deps --user 0 --entrypoint sh todoapp-postgres \
			-c 'find /var/lib/postgresql -mindepth 1 -delete' && \
		$(COMPOSE) --profile tools down; \
	else \
		printf 'Очистка отменена.\n'; \
	fi

env-port-forward: out/go.mod ## Открыть PostgreSQL на localhost:5432
	@$(COMPOSE) up -d --wait port-forwarder

env-port-close: ## Закрыть проброс, оставив PostgreSQL работающим
	@$(COMPOSE) stop port-forwarder
	@$(COMPOSE) rm -f port-forwarder

migrate-create: ## Создать пару миграций: make migrate-create seq=add_users
	@test -n "$(seq)" || { printf 'Укажите имя миграции: make migrate-create seq=init\n' >&2; exit 1; }
	@$(COMPOSE) run --rm --no-deps --user "$$(id -u):$$(id -g)" todoapp-postgres-migrate \
		create -ext sql -dir /migrations -seq "$(seq)"

migrate-up: ## Применить все новые миграции
	@$(MIGRATE) up

migrate-down: ## Откатить последнюю миграцию (удаляет её данные)
	@$(MIGRATE) down 1

# Keep Go package discovery out of PostgreSQL data files.
out/go.mod:
	@mkdir -p out
	@printf "module local.todoapp.data\n" > $@

.PHONY: run build test vet test-integration

run: out/go.mod ## Запустить HTTP API локально
	@POSTGRES_HOST=localhost go run ./cmd/todoapp

build: out/go.mod ## Собрать приложение в out/todoapp
	@go build -o out/todoapp ./cmd/todoapp

test: out/go.mod ## Запустить Go-тесты с проверкой гонок
	@go test -race ./...

vet: out/go.mod ## Статическая проверка Go
	@go vet ./...

test-integration: out/go.mod ## Прогнать API на отдельном временном PostgreSQL
	@sh scripts/test-integration.sh
