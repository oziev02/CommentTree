# Быстрая справка по архитектуре CommentTree

## Структура слоев

```
┌─────────────────────────────────────────┐
│  Delivery (HTTP)                        │  ← Внешний слой
│  - handlers, middleware, router         │
├─────────────────────────────────────────┤
│  Use Case                               │  ← Бизнес-логика
│  - валидация, бизнес-правила            │
├─────────────────────────────────────────┤
│  Domain                                 │  ← Ядро (независимо)
│  - модели, интерфейсы, ошибки           │
├─────────────────────────────────────────┤
│  Infrastructure                         │  ← Реализации
│  - PostgreSQL, миграции                 │
└─────────────────────────────────────────┘
```

## Направление зависимостей

```
main.go
  ↓
  ├─→ config          (конфигурация)
  ├─→ infrastructure   (реализация репозитория)
  ├─→ usecase          (бизнес-логика)
  │     ↓
  │     └─→ domain     (интерфейсы)
  │
  └─→ delivery         (HTTP handlers)
        ↓
        └─→ usecase    (бизнес-логика)
              ↓
              └─→ domain (интерфейсы)
                    ↓
                    └─→ infrastructure (реализация)
```

**Правило**: Зависимости направлены внутрь. Domain не зависит ни от чего.

## Компоненты по слоям

### Domain Layer
- **Файлы**: `internal/domain/comment.go`, `errors.go`
- **Содержит**: 
  - `Comment` - модель комментария
  - `CommentTree` - дерево комментариев
  - `CommentFilter` - фильтры и пагинация
  - `CommentRepository` - интерфейс репозитория
  - Доменные ошибки

### Use Case Layer
- **Файлы**: `internal/usecase/comment.go`
- **Содержит**:
  - `CommentUseCase` - бизнес-логика
  - Методы: `Create()`, `GetTree()`, `Delete()`, `GetTotalCount()`
- **Зависит от**: `domain.CommentRepository` (интерфейс)

### Infrastructure Layer
- **Файлы**: `internal/infrastructure/database/postgres.go`
- **Содержит**:
  - `PostgresRepository` - реализация репозитория
  - Рекурсивные SQL запросы
  - Миграции БД
- **Реализует**: `domain.CommentRepository`

### Delivery Layer
- **Файлы**: `internal/delivery/http/*.go`
- **Содержит**:
  - `CommentHandler` - HTTP handlers
  - `NewRouter()` - маршрутизация
  - `LoggingMiddleware`, `CORSMiddleware`
- **Зависит от**: `usecase.CommentUseCase`

### Config Layer
- **Файлы**: `internal/config/config.go`
- **Содержит**:
  - `Config` - структура конфигурации
  - `Load()` - загрузка из env переменных

## Основные потоки

### Создание комментария
```
HTTP POST /comments
  → Handler.Create()
    → UseCase.Create()
      → Repo.GetByID() (проверка родителя)
      → Repo.Create()
  → JSON Response
```

### Получение дерева
```
HTTP GET /comments?page=1
  → Handler.GetTree()
    → UseCase.GetTree()
      → Repo.GetTree() (рекурсивный SQL)
  → JSON Response с деревом
```

### Удаление комментария
```
HTTP DELETE /comments/{id}
  → Handler.Delete()
    → UseCase.Delete()
      → Repo.GetByID() (проверка существования)
      → Repo.Delete() (рекурсивное удаление)
  → 204 No Content
```

## Ключевые принципы

1. **Dependency Inversion**: Use case зависит от интерфейса, а не от реализации
2. **Single Responsibility**: Каждый слой решает свою задачу
3. **Separation of Concerns**: Бизнес-логика отделена от инфраструктуры
4. **Testability**: Легко мокировать интерфейсы для тестов

## Где что искать

| Задача | Где искать |
|--------|-----------|
| Изменить бизнес-правила | `internal/usecase/` |
| Изменить структуру данных | `internal/domain/` |
| Изменить SQL запросы | `internal/infrastructure/database/` |
| Добавить новый API endpoint | `internal/delivery/http/` |
| Изменить конфигурацию | `internal/config/` |
| Добавить middleware | `internal/delivery/http/middleware.go` |
| Изменить точку входа | `cmd/app/main.go` |

## Расширение проекта

### Добавить новый use case
1. Создать файл в `internal/usecase/`
2. Использовать интерфейсы из `domain`
3. Добавить handler в `delivery/http/`

### Добавить новый delivery слой (gRPC)
1. Создать `internal/delivery/grpc/`
2. Использовать те же use cases
3. Инициализировать в `main.go`

### Заменить БД
1. Создать новую реализацию `domain.CommentRepository`
2. Изменить инициализацию в `main.go`
3. Use cases остаются без изменений

