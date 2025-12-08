# CommentTree

Сервис для работы с древовидными комментариями с поддержкой неограниченной вложенности, поиска, сортировки и постраничного вывода.

## Технологии

- **Go 1.22** - язык программирования
- **PostgreSQL** - база данных с поддержкой рекурсивных запросов
- **pgx/v5** - драйвер PostgreSQL для Go
- **godotenv** - загрузка переменных окружения из `.env` файла
- **slog** - структурированное логирование (стандартная библиотека Go)
- **net/http** - HTTP сервер (стандартная библиотека Go, без фреймворков)

## Архитектура

Проект реализован с использованием Standard Go Project Layout и Clean Architecture:

- `cmd/app` - точка входа приложения
- `internal/domain` - доменные модели и интерфейсы
- `internal/usecase` - бизнес-логика
- `internal/infrastructure` - реализация репозиториев (PostgreSQL)
- `internal/delivery` - HTTP handlers и middleware
- `internal/config` - конфигурация приложения
- `web` - веб-интерфейс
- `deployments` - конфигурации для развертывания

**Документация по архитектуре**:
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - подробное описание архитектуры
- [FLOW.md](docs/FLOW.md) - потоки данных и взаимодействие компонентов
- [DIAGRAMS.md](docs/DIAGRAMS.md) - визуальные диаграммы архитектуры
- [QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md) - быстрая справка

## Требования

- Go 1.22 или выше
- PostgreSQL 12 или выше
- Docker и Docker Compose (опционально, для быстрого запуска)

## Установка и запуск

### Быстрый старт с Makefile

Показать все доступные команды:
```bash
make help
```

### С использованием Docker Compose

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd CommentTree
```

2. Запустите сервисы:
```bash
make docker-up
```

Или вручную:
```bash
cd deployments
docker-compose up -d
```

Это запустит PostgreSQL и приложение. Приложение будет доступно по адресу http://localhost:8080

Остановить контейнеры:
```bash
make docker-down
```

Просмотр логов:
```bash
make docker-logs
```

### Ручная установка

1. Установите зависимости:
```bash
make install
```

Или вручную:
```bash
go mod download
```

2. Создайте базу данных PostgreSQL:
```bash
createdb commenttree
```

3. Примените миграции:
```bash
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=commenttree make migrate-up
```

Или вручную:
```bash
psql -d commenttree -f internal/infrastructure/database/migrations/001_create_comments.up.sql
```

4. Настройте переменные окружения (опционально):
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=commenttree
export SERVER_HOST=localhost
export SERVER_PORT=8080
```

5. Запустите приложение:
```bash
make run
```

Или вручную:
```bash
go run ./cmd/app
```

### Сборка приложения

Собрать бинарный файл:
```bash
make build
```

Результат: `./commenttree`

### Разработка

Запуск в режиме разработки (с автоперезагрузкой, требует air):
```bash
make dev
```

Форматирование кода:
```bash
make fmt
```

Проверка кода:
```bash
make vet
```

Запуск всех проверок:
```bash
make check
```

Полная сборка (очистка, зависимости, проверки, сборка):
```bash
make all
```


## Особенности реализации

### Graceful Shutdown

Приложение поддерживает корректное завершение работы:
- Обработка сигналов SIGINT и SIGTERM
- Таймаут завершения: 30 секунд
- Корректное закрытие соединений с БД

### Логирование

Используется структурированное логирование через `slog`:
- JSON формат для продакшена
- Логирование всех HTTP запросов через middleware
- Логирование ошибок с контекстом

### Без фреймворков

Проект использует только стандартную библиотеку Go:
- `net/http` для HTTP сервера
- `slog` для логирования
- Минимальные внешние зависимости (только pgx и godotenv)

## API

### POST /comments

Создает новый комментарий.

Запрос:
```json
{
  "parent_id": 1,
  "content": "Текст комментария"
}
```

Ответ:
```json
{
  "id": 1,
  "parent_id": null,
  "content": "Текст комментария",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

### GET /comments

Получает дерево комментариев с поддержкой фильтрации и пагинации.

Параметры запроса:
- `parent` (опционально) - ID родительского комментария
- `search` (опционально) - поисковый запрос
- `page` (опционально) - номер страницы (по умолчанию 1)
- `page_size` (опционально) - размер страницы (по умолчанию 50)
- `sort_by` (опционально) - поле сортировки: `created_at` или `updated_at` (по умолчанию `created_at`)
- `order` (опционально) - порядок сортировки: `asc` или `desc` (по умолчанию `desc`)

Пример:
```
GET /comments?page=1&page_size=20&sort_by=created_at&order=desc
```

Ответ:
```json
{
  "comments": [
    {
      "comment": {
        "id": 1,
        "content": "Комментарий 1",
        "created_at": "2024-01-01T12:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
      },
      "children": [
        {
          "comment": {
            "id": 2,
            "parent_id": 1,
            "content": "Ответ 1.1",
            "created_at": "2024-01-01T12:05:00Z",
            "updated_at": "2024-01-01T12:05:00Z"
          },
          "children": []
        }
      ]
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

### DELETE /comments/{id}

Удаляет комментарий и все вложенные комментарии.

Ответ: 204 No Content

## Web интерфейс

После запуска приложения веб-интерфейс доступен по адресу http://localhost:8080

Функции интерфейса:
- Просмотр дерева комментариев с визуальной вложенностью
- Создание новых комментариев и ответов
- Удаление комментариев
- Поиск комментариев по ключевым словам
- Постраничная навигация

## Конфигурация

Приложение использует переменные окружения для конфигурации. Поддерживается загрузка из `.env` файла для удобства локальной разработки.

### Приоритет загрузки конфигурации

1. **Переменные окружения системы** (высший приоритет)
2. **`.env` файл** (если существует)
3. **Значения по умолчанию**

### Настройка через .env файл (рекомендуется для локальной разработки)

1. Скопируйте шаблон:
```bash
cp .env.example .env
```

2. Отредактируйте `.env` файл и укажите свои значения:
```bash
SERVER_HOST=localhost
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=commenttree
DB_SSLMODE=disable
```

3. Запустите приложение - переменные загрузятся автоматически.

### Настройка через переменные окружения

Вы можете установить переменные окружения напрямую:

```bash
export SERVER_HOST=localhost
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=commenttree
export DB_SSLMODE=disable
```

### Переменные конфигурации

- `SERVER_HOST` - хост HTTP сервера (по умолчанию: localhost)
- `SERVER_PORT` - порт HTTP сервера (по умолчанию: 8080)
- `DB_HOST` - хост PostgreSQL (по умолчанию: localhost)
- `DB_PORT` - порт PostgreSQL (по умолчанию: 5432)
- `DB_USER` - пользователь PostgreSQL (по умолчанию: postgres)
- `DB_PASSWORD` - пароль PostgreSQL (по умолчанию: postgres)
- `DB_NAME` - имя базы данных (по умолчанию: commenttree)
- `DB_SSLMODE` - режим SSL (по умолчанию: disable)

**Важно**: Файл `.env` уже добавлен в `.gitignore` и не будет закоммичен в репозиторий. Используйте `.env.example` как шаблон.

## Разработка

### Структура проекта

```
CommentTree/
├── cmd/
│   └── app/           # Точка входа приложения
│       └── main.go
├── internal/          # Приватный код приложения
│   ├── domain/       # Доменные модели и интерфейсы
│   │   ├── comment.go
│   │   └── errors.go
│   ├── usecase/      # Бизнес-логика
│   │   └── comment.go
│   ├── infrastructure/ # Инфраструктура
│   │   └── database/
│   │       ├── postgres.go
│   │       └── migrations/
│   ├── delivery/     # HTTP handlers
│   │   └── http/
│   │       ├── handler.go
│   │       ├── middleware.go
│   │       └── router.go
│   └── config/       # Конфигурация
│       └── config.go
├── web/              # Веб-интерфейс
│   └── index.html
├── deployments/     # Конфигурации для развертывания
│   └── docker-compose.yml
├── Dockerfile
└── go.mod
```

## Зависимости

Основные зависимости проекта (см. `go.mod`):

- `github.com/jackc/pgx/v5` - драйвер PostgreSQL
- `github.com/joho/godotenv` - загрузка переменных окружения

Все зависимости управляются через Go modules.

## Лицензия

MIT

