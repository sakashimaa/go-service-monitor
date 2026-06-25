-include .env
export

APP_NAME = site-monitor
MAIN_PATH = cmd/monitor/main.go
MIGRATIONS_DIR = ./migrations
VERSION ?= v1.0.0-local

.PHONY: help build docker-build clean run up down restart deps fmt lint test swag migrate-up migrate-down db-reset logs ps shell

# генерация документации по регуляркам. Для документации команды обязательно должно быть вот так <название_команды>: ## документация
help: ## список всех доступных команд
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## локальная сборка бинарника
	go build -ldflags="-w -s" -o bin/$(APP_NAME) $(MAIN_PATH)

docker-build: ## сборка докер
	docker build --build-arg VERSION=$(VERSION) -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

clean: ## очистка артефактов сборки
	rm -rf bin/

run: swag ## запуск с авто генерацией доки свагер
	go run $(MAIN_PATH)

up: ## запуск и сборка контейнеров
	docker compose up --build -d

down: ## остановка контейнеров
	docker compose down

restart: ## перезапуск контейнеров
	docker compose restart

deps: ## загрузка зависимостей локально
	go mod download
	go mod tidy

fmt: ## форматирование всего проекта
	go fmt ./...

lint: ## проверка линтерос
	golangci-lint run

test: ## запуск тестов (пока тестов нет)
	go test -v ./...

swag: ## генерация документации Swagger
	$(shell go env GOPATH)/bin/swag init -g $(MAIN_PATH)

migrate-up: ## мигрировать базу (пока миграций нет)
	@echo "Running migrations up..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down: ## откатить миграции базы (пока миграций нет)
	@echo "Running migrations down..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

db-reset: ## удаление данных БД и перезапуск контейнера
	docker compose down -v
	docker compose up -d postgres

logs: ## просмотр логов контейнеров
	docker compose logs -f

ps: ## просмотр статуса контейнеров
	docker ps

shell: ## запуск интерактивной консоли внутри контейнера приложения
	docker exec -it monitor-app /bin/sh