.PHONY: help build run clean docker-build docker-up docker-down migrate-up migrate-down lint fmt vet deps install env dev check all info

# Переменные
APP_NAME := commenttree
BINARY_NAME := $(APP_NAME)
CMD_PATH := ./cmd/app
DOCKER_COMPOSE := docker-compose
DOCKER_COMPOSE_FILE := deployments/docker-compose.yml
MIGRATIONS_PATH := internal/infrastructure/database/migrations

# Цвета для вывода
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

help: ## Показать справку по командам
	@echo "$(YELLOW)Доступные команды:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

deps: ## Загрузить зависимости
	@echo "$(YELLOW)Загрузка зависимостей...$(RESET)"
	go mod download
	go mod tidy

env: ## Создать .env файл из примера
	@if [ -f .env ]; then \
		echo "$(YELLOW).env файл уже существует$(RESET)"; \
	else \
		cp .env.example .env; \
		echo "$(GREEN).env файл создан из .env.example$(RESET)"; \
		echo "$(YELLOW)Отредактируйте .env файл и укажите свои значения$(RESET)"; \
	fi

install: deps ## Установить зависимости и проверить код
	@echo "$(YELLOW)Установка зависимостей...$(RESET)"
	go mod download
	go mod verify

build: ## Собрать приложение
	@echo "$(YELLOW)Сборка приложения...$(RESET)"
	go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "$(GREEN)Сборка завершена: $(BINARY_NAME)$(RESET)"

run: ## Запустить приложение локально
	@echo "$(YELLOW)Запуск приложения...$(RESET)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Предупреждение: .env файл не найден. Используются значения по умолчанию или переменные окружения.$(RESET)"; \
		echo "$(YELLOW)Создайте .env файл из шаблона: cp .env.example .env$(RESET)"; \
	fi
	go run $(CMD_PATH)

lint: ## Запустить линтер
	@echo "$(YELLOW)Проверка кода линтером...$(RESET)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint не установлен, пропускаем...$(RESET)"; \
	fi

fmt: ## Форматировать код
	@echo "$(YELLOW)Форматирование кода...$(RESET)"
	go fmt ./...
	@echo "$(GREEN)Код отформатирован$(RESET)"

vet: ## Проверить код с помощью go vet
	@echo "$(YELLOW)Проверка кода go vet...$(RESET)"
	go vet ./...
	@echo "$(GREEN)Проверка завершена$(RESET)"

clean: ## Очистить артефакты сборки
	@echo "$(YELLOW)Очистка...$(RESET)"
	rm -f $(BINARY_NAME)
	go clean -cache
	@echo "$(GREEN)Очистка завершена$(RESET)"

# Docker команды
docker-build: ## Собрать Docker образ
	@echo "$(YELLOW)Сборка Docker образа...$(RESET)"
	docker build -t $(APP_NAME):latest .
	@echo "$(GREEN)Docker образ собран$(RESET)"

docker-up: ## Запустить контейнеры через docker-compose
	@echo "$(YELLOW)Запуск контейнеров...$(RESET)"
	cd deployments && $(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Контейнеры запущены$(RESET)"

docker-down: ## Остановить контейнеры
	@echo "$(YELLOW)Остановка контейнеров...$(RESET)"
	cd deployments && $(DOCKER_COMPOSE) down
	@echo "$(GREEN)Контейнеры остановлены$(RESET)"

docker-logs: ## Показать логи контейнеров
	cd deployments && $(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Перезапустить контейнеры

docker-clean: docker-down ## Остановить и удалить контейнеры и volumes
	@echo "$(YELLOW)Удаление контейнеров и volumes...$(RESET)"
	cd deployments && $(DOCKER_COMPOSE) down -v
	@echo "$(GREEN)Контейнеры и volumes удалены$(RESET)"

# Миграции
migrate-up: ## Применить миграции БД
	@echo "$(YELLOW)Применение миграций...$(RESET)"
	@if [ -z "$$DB_HOST" ]; then \
		echo "$(YELLOW)Используется локальная БД. Установите переменные окружения:$(RESET)"; \
		echo "  DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=commenttree make migrate-up"; \
		exit 1; \
	fi
	psql -h $$DB_HOST -p $$DB_PORT -U $$DB_USER -d $$DB_NAME -f $(MIGRATIONS_PATH)/001_create_comments.up.sql
	@echo "$(GREEN)Миграции применены$(RESET)"

migrate-down: ## Откатить миграции БД
	@echo "$(YELLOW)Откат миграций...$(RESET)"
	@if [ -z "$$DB_HOST" ]; then \
		echo "$(YELLOW)Используется локальная БД. Установите переменные окружения:$(RESET)"; \
		echo "  DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=commenttree make migrate-down"; \
		exit 1; \
	fi
	psql -h $$DB_HOST -p $$DB_PORT -U $$DB_USER -d $$DB_NAME -f $(MIGRATIONS_PATH)/001_create_comments.down.sql
	@echo "$(GREEN)Миграции откачены$(RESET)"

# Разработка
dev: ## Запустить в режиме разработки (с автоперезагрузкой, требует air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(YELLOW)air не установлен. Установите: go install github.com/cosmtrek/air@latest$(RESET)"; \
		echo "$(YELLOW)Или используйте: make run$(RESET)"; \
	fi

# Проверка перед коммитом
check: fmt vet ## Запустить все проверки (fmt, vet)
	@echo "$(GREEN)Все проверки пройдены$(RESET)"

# Полная сборка и проверка
all: clean deps fmt vet build ## Полная сборка: очистка, зависимости, проверки, сборка
	@echo "$(GREEN)Полная сборка завершена$(RESET)"

# Информация о проекте
info: ## Показать информацию о проекте
	@echo "$(YELLOW)Информация о проекте:$(RESET)"
	@echo "  Имя приложения: $(APP_NAME)"
	@echo "  Путь к main: $(CMD_PATH)"
	@echo "  Go версия: $(shell go version)"
	@echo "  Go модуль: $(shell head -1 go.mod | cut -d' ' -f2)"

