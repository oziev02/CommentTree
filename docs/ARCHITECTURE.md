# Архитектура проекта CommentTree

## Обзор

Проект использует комбинацию **Standard Go Project Layout** и **Clean Architecture** для создания масштабируемого и поддерживаемого сервиса древовидных комментариев.

## Технологии

- **Go 1.22** - язык программирования
- **PostgreSQL** - база данных с поддержкой рекурсивных запросов (WITH RECURSIVE)
- **pgx/v5** - драйвер PostgreSQL для Go (высокая производительность, пул соединений)
- **godotenv** - загрузка переменных окружения из `.env` файла
- **slog** - структурированное логирование (стандартная библиотека Go)
- **net/http** - HTTP сервер (стандартная библиотека Go, без внешних фреймворков)

Проект сознательно использует минимальный набор зависимостей, полагаясь на стандартную библиотеку Go для максимальной производительности и простоты поддержки.

## Принципы архитектуры

### Clean Architecture

Архитектура разделена на слои с четкими границами и зависимостями:

```
┌─────────────────────────────────────────┐
│         Delivery Layer (HTTP)          │  ← Внешний слой
├─────────────────────────────────────────┤
│         Use Case Layer                 │  ← Бизнес-логика
├─────────────────────────────────────────┤
│         Domain Layer                   │  ← Ядро (независимо)
├─────────────────────────────────────────┤
│         Infrastructure Layer           │  ← Реализации
└─────────────────────────────────────────┘
```

**Правило зависимостей**: Внутренние слои не зависят от внешних. Все зависимости направлены внутрь.

### Standard Go Project Layout

Структура директорий следует стандартным практикам Go:

- `cmd/` - точки входа приложения
- `internal/` - приватный код (не импортируется извне)
- `web/` - статические файлы
- `deployments/` - конфигурации развертывания

## Структура проекта

```
CommentTree/
├── cmd/
│   └── app/
│       └── main.go                    # Точка входа, инициализация
│
├── internal/                          # Приватный код приложения
│   ├── domain/                        # Доменный слой (ядро)
│   │   ├── comment.go                 # Доменные модели
│   │   └── errors.go                  # Доменные ошибки
│   │
│   ├── usecase/                       # Слой бизнес-логики
│   │   └── comment.go                 # Use cases для комментариев
│   │
│   ├── infrastructure/                # Слой инфраструктуры
│   │   └── database/
│   │       ├── postgres.go            # Реализация репозитория
│   │       └── migrations/            # SQL миграции
│   │
│   ├── delivery/                      # Слой доставки (HTTP)
│   │   └── http/
│   │       ├── handler.go            # HTTP handlers
│   │       ├── middleware.go         # HTTP middleware
│   │       └── router.go             # Маршрутизация
│   │
│   └── config/                        # Конфигурация
│       └── config.go                 # Загрузка конфигурации
│
├── web/                               # Веб-интерфейс
│   └── index.html
│
└── deployments/                       # Конфигурации развертывания
    └── docker-compose.yml
```

## Детальное описание слоев

### 1. Domain Layer (Доменный слой)

**Назначение**: Содержит бизнес-сущности и интерфейсы. Это ядро приложения, не зависящее от внешних библиотек.

**Файлы**:
- `internal/domain/comment.go` - модели данных и интерфейсы
- `internal/domain/errors.go` - доменные ошибки

**Компоненты**:

#### Модели данных

```go
// Comment - основная сущность комментария
type Comment struct {
    ID        int64
    ParentID  *int64    // nil для корневых комментариев
    Content   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// CommentTree - комментарий со вложенными комментариями
type CommentTree struct {
    Comment  Comment
    Children []CommentTree
}

// CommentFilter - параметры фильтрации и пагинации
type CommentFilter struct {
    ParentID *int64
    Search   string
    Page     int
    PageSize int
    SortBy   string
    Order    string
}
```

#### Интерфейсы (Dependency Inversion)

```go
// CommentRepository - интерфейс для работы с данными
// Определен в domain, реализован в infrastructure
type CommentRepository interface {
    Create(comment *Comment) error
    GetByID(id int64) (*Comment, error)
    GetTree(parentID *int64, filter CommentFilter) ([]CommentTree, error)
    Delete(id int64) error
    Search(query string, filter CommentFilter) ([]CommentTree, error)
    Count(parentID *int64, search string) (int, error)
}
```

**Принципы**:
- Не зависит от внешних библиотек
- Содержит только бизнес-логику и интерфейсы
- Определяет контракты для других слоев

### 2. Use Case Layer (Слой бизнес-логики)

**Назначение**: Содержит бизнес-логику приложения. Оркестрирует работу доменных моделей.

**Файлы**:
- `internal/usecase/comment.go`

**Компоненты**:

```go
type CommentUseCase struct {
    repo domain.CommentRepository  // Зависит от интерфейса, не от реализации
}

// Методы use case:
// - Create() - создание комментария с валидацией
// - GetTree() - получение дерева комментариев
// - Delete() - удаление комментария и поддерева
// - GetTotalCount() - подсчет комментариев
```

**Ответственность**:
- Валидация входных данных
- Бизнес-правила (например, проверка существования родителя)
- Координация работы с репозиторием
- Обработка ошибок доменного слоя

**Пример потока**:

```
Create(comment) →
  1. Валидация (пустой контент?)
  2. Проверка родителя (если указан)
  3. Создание через репозиторий
  4. Возврат результата
```

### 3. Infrastructure Layer (Слой инфраструктуры)

**Назначение**: Реализует интерфейсы, определенные в domain. Работает с внешними системами (БД, API и т.д.).

**Файлы**:
- `internal/infrastructure/database/postgres.go` - реализация репозитория
- `internal/infrastructure/database/migrations/` - SQL миграции

**Компоненты**:

```go
type PostgresRepository struct {
    pool *pgxpool.Pool  // Зависит от pgx (внешняя библиотека)
}

// Реализует domain.CommentRepository
func (r *PostgresRepository) Create(comment *domain.Comment) error { ... }
func (r *PostgresRepository) GetByID(id int64) (*domain.Comment, error) { ... }
// ... и другие методы интерфейса
```

**Особенности**:
- Использует рекурсивные SQL запросы (WITH RECURSIVE) для работы с деревом
- Реализует полнотекстовый поиск
- Обрабатывает пагинацию на уровне БД

**Миграции**:
- `001_create_comments.up.sql` - создание таблицы и индексов
- `001_create_comments.down.sql` - откат миграции

### 4. Delivery Layer (Слой доставки)

**Назначение**: Обрабатывает HTTP запросы, преобразует их в вызовы use cases, форматирует ответы.

**Файлы**:
- `internal/delivery/http/handler.go` - обработчики запросов
- `internal/delivery/http/router.go` - маршрутизация
- `internal/delivery/http/middleware.go` - middleware (логирование, CORS)

**Компоненты**:

#### Router
```go
// Определяет маршруты:
// POST   /comments      → Create
// GET    /comments      → GetTree
// DELETE /comments/{id} → Delete
```

#### Handler
```go
type CommentHandler struct {
    useCase *usecase.CommentUseCase  // Зависит от use case
}

// Методы:
// - Create() - парсит JSON, вызывает use case, возвращает JSON
// - GetTree() - парсит query параметры, вызывает use case
// - Delete() - извлекает ID из пути, вызывает use case
```

#### DTO (Data Transfer Objects)
```go
// CreateCommentRequest - входной DTO
type CreateCommentRequest struct {
    ParentID *int64 `json:"parent_id"`
    Content  string `json:"content"`
}

// CommentResponse - выходной DTO
type CommentResponse struct {
    ID        int64  `json:"id"`
    ParentID  *int64 `json:"parent_id,omitempty"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

**Принципы**:
- Тонкие handlers (только валидация, вызов use case, форматирование)
- Использование DTO вместо доменных моделей в API
- Обработка HTTP-специфичных ошибок

### 5. Config Layer (Слой конфигурации)

**Назначение**: Загружает и предоставляет конфигурацию приложения.

**Файлы**:
- `internal/config/config.go`

**Компоненты**:
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
}

// Загружает из переменных окружения
func Load() (*Config, error)
```

## Поток данных

### Пример: Создание комментария

```
1. HTTP Request
   POST /comments
   Body: {"parent_id": 1, "content": "Текст"}

2. Delivery Layer (handler.go)
   ├─ Парсинг JSON → CreateCommentRequest
   ├─ Вызов: useCase.Create(ctx, req.ParentID, req.Content)
   └─ Преобразование результата → CommentResponse

3. Use Case Layer (usecase/comment.go)
   ├─ Валидация: content != ""
   ├─ Проверка родителя: repo.GetByID(parentID)
   ├─ Создание: repo.Create(comment)
   └─ Возврат: *domain.Comment

4. Infrastructure Layer (database/postgres.go)
   ├─ SQL запрос: INSERT INTO comments ...
   ├─ Получение ID: RETURNING id
   └─ Возврат: error

5. HTTP Response
   201 Created
   Body: {"id": 1, "content": "Текст", ...}
```

### Пример: Получение дерева комментариев

```
1. HTTP Request
   GET /comments?page=1&page_size=20&sort_by=created_at&order=desc

2. Delivery Layer
   ├─ Парсинг query параметров → CommentFilter
   ├─ Вызов: useCase.GetTree(ctx, filter)
   └─ Преобразование → CommentsListResponse

3. Use Case Layer
   ├─ Нормализация параметров (значения по умолчанию)
   ├─ Вызов: repo.GetTree(filter.ParentID, filter)
   └─ Возврат: []domain.CommentTree

4. Infrastructure Layer
   ├─ Рекурсивный SQL запрос (WITH RECURSIVE)
   ├─ Построение дерева в памяти
   └─ Возврат: []domain.CommentTree

5. HTTP Response
   200 OK
   Body: {"comments": [...], "total": 10, "page": 1, "page_size": 20}
```

## Зависимости между слоями

```
main.go
  ↓
  ├─→ config.Load()                    [config]
  ├─→ database.NewPostgresRepository() [infrastructure]
  ├─→ usecase.NewCommentUseCase()      [usecase]
  │     ↓
  │     └─→ domain.CommentRepository    [domain - интерфейс]
  │
  └─→ http.NewRouter()                 [delivery]
        ↓
        └─→ usecase.CommentUseCase     [usecase]
              ↓
              └─→ domain.CommentRepository [domain - интерфейс]
                    ↓
                    └─→ PostgresRepository [infrastructure - реализация]
```

**Ключевые принципы**:
- `main.go` знает о всех слоях (инициализация)
- `delivery` зависит от `usecase`
- `usecase` зависит от `domain` (интерфейсы)
- `infrastructure` реализует интерфейсы из `domain`
- `domain` не зависит ни от чего

## Инициализация приложения (main.go)

```go
1. Загрузка конфигурации
   config.Load()

2. Подключение к БД
   pgxpool.New(cfg.Database.DSN())

3. Создание репозитория
   database.NewPostgresRepository(pool)

4. Создание use case
   usecase.NewCommentUseCase(repo)

5. Создание роутера
   http.NewRouter(commentUseCase)

6. Настройка middleware
   CORSMiddleware → LoggingMiddleware

7. Запуск HTTP сервера
   server.ListenAndServe()
```

## Преимущества архитектуры

### 1. Тестируемость
- Легко мокировать интерфейсы репозитория
- Use cases можно тестировать без БД
- Handlers можно тестировать с моками use cases

### 2. Независимость от фреймворков
- Domain и Use Case не зависят от HTTP или БД
- Можно легко добавить gRPC, CLI или другой delivery слой
- Можно заменить PostgreSQL на другую БД

### 3. Поддерживаемость
- Четкое разделение ответственности
- Легко найти код по функциональности
- Изменения в одном слое не влияют на другие

### 4. Масштабируемость
- Легко добавлять новые use cases
- Легко добавлять новые handlers
- Легко добавлять новые реализации репозиториев

## Расширение архитектуры

### Добавление нового use case

1. Добавить метод в `usecase/comment.go` или создать новый файл
2. Добавить обработчик в `delivery/http/handler.go`
3. Добавить маршрут в `delivery/http/router.go`

### Добавление нового delivery слоя (например, gRPC)

1. Создать `internal/delivery/grpc/`
2. Реализовать gRPC handlers, использующие те же use cases
3. Добавить инициализацию в `main.go`

### Замена БД

1. Создать новую реализацию `domain.CommentRepository`
2. В `main.go` использовать новую реализацию
3. Use cases остаются без изменений

## Заключение

Архитектура обеспечивает:
- **Разделение ответственности** - каждый слой решает свою задачу
- **Независимость** - внутренние слои не зависят от внешних
- **Тестируемость** - легко тестировать каждый слой отдельно
- **Гибкость** - легко менять реализации без изменения бизнес-логики

Это позволяет создавать надежные, поддерживаемые и масштабируемые приложения.

