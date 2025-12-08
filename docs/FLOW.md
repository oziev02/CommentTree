# Потоки данных в CommentTree

## Визуализация потоков

### 1. Создание комментария

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /comments
       │ {"parent_id": 1, "content": "Текст"}
       ▼
┌─────────────────────────────────────┐
│   Delivery Layer (HTTP Handler)     │
│   ────────────────────────────────  │
│   1. Парсинг JSON                   │
│   2. Валидация структуры             │
│   3. Вызов useCase.Create()         │
│   4. Преобразование в DTO           │
│   5. Возврат JSON ответа            │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Use Case Layer                    │
│   ────────────────────────────────  │
│   1. Валидация: content != ""        │
│   2. Проверка родителя (если есть)   │
│      └─→ repo.GetByID(parentID)     │
│   3. Создание комментария           │
│      └─→ repo.Create(comment)      │
│   4. Возврат domain.Comment         │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Infrastructure Layer (Postgres)  │
│   ────────────────────────────────  │
│   1. SQL: INSERT INTO comments ...  │
│   2. Получение ID: RETURNING id     │
│   3. Возврат error (или nil)        │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│  PostgreSQL │
└─────────────┘
```

**Код по слоям**:

```go
// Delivery: handler.go
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateCommentRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    comment, err := h.useCase.Create(r.Context(), req.ParentID, req.Content)
    // обработка ошибок...
    
    json.NewEncoder(w).Encode(toCommentResponse(comment))
}

// Use Case: usecase/comment.go
func (uc *CommentUseCase) Create(ctx context.Context, parentID *int64, content string) (*domain.Comment, error) {
    if content == "" {
        return nil, domain.ErrEmptyContent
    }
    
    if parentID != nil {
        parent, err := uc.repo.GetByID(*parentID)
        // проверка...
    }
    
    comment := &domain.Comment{ParentID: parentID, Content: content}
    err := uc.repo.Create(comment)
    
    return comment, err
}

// Infrastructure: database/postgres.go
func (r *PostgresRepository) Create(comment *domain.Comment) error {
    query := `INSERT INTO comments ... RETURNING id`
    return r.pool.QueryRow(ctx, query, ...).Scan(&comment.ID)
}
```

### 2. Получение дерева комментариев

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ GET /comments?page=1&parent=5
       ▼
┌─────────────────────────────────────┐
│   Delivery Layer                    │
│   ────────────────────────────────  │
│   1. Парсинг query параметров       │
│   2. Создание CommentFilter         │
│   3. Вызов useCase.GetTree()        │
│   4. Получение total count          │
│   5. Преобразование в DTO           │
│   6. Возврат JSON с пагинацией      │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Use Case Layer                    │
│   ────────────────────────────────  │
│   1. Нормализация параметров       │
│      (page, pageSize, sortBy, order)│
│   2. Вызов repo.GetTree()           │
│   3. Возврат []CommentTree           │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Infrastructure Layer              │
│   ────────────────────────────────  │
│   1. Рекурсивный SQL запрос:        │
│      WITH RECURSIVE comment_tree... │
│   2. Получение всех комментариев    │
│   3. Построение дерева в памяти     │
│      (buildTree рекурсивно)         │
│   4. Возврат []CommentTree           │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│  PostgreSQL │
└─────────────┘
```

**Рекурсивный SQL запрос**:

```sql
WITH RECURSIVE comment_tree AS (
    -- Базовый случай: корневые комментарии
    SELECT id, parent_id, content, created_at, updated_at, 0 as level
    FROM comments
    WHERE parent_id IS NULL
    
    UNION ALL
    
    -- Рекурсивный случай: дочерние комментарии
    SELECT c.id, c.parent_id, c.content, c.created_at, c.updated_at, ct.level + 1
    FROM comments c
    INNER JOIN comment_tree ct ON c.parent_id = ct.id
)
SELECT * FROM comment_tree WHERE level = 0
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;
```

### 3. Удаление комментария

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ DELETE /comments/5
       ▼
┌─────────────────────────────────────┐
│   Delivery Layer                    │
│   ────────────────────────────────  │
│   1. Извлечение ID из пути          │
│   2. Вызов useCase.Delete(id)       │
│   3. Возврат 204 No Content          │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Use Case Layer                    │
│   ────────────────────────────────  │
│   1. Проверка существования         │
│      └─→ repo.GetByID(id)          │
│   2. Удаление поддерева             │
│      └─→ repo.Delete(id)           │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Infrastructure Layer              │
│   ────────────────────────────────  │
│   1. Рекурсивный SQL для поиска     │
│      всех потомков                  │
│   2. DELETE всех найденных          │
│      комментариев                  │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│  PostgreSQL │
└─────────────┘
```

**Рекурсивное удаление**:

```sql
WITH RECURSIVE comment_tree AS (
    SELECT id FROM comments WHERE id = $1
    UNION ALL
    SELECT c.id FROM comments c
    INNER JOIN comment_tree ct ON c.parent_id = ct.id
)
DELETE FROM comments WHERE id IN (SELECT id FROM comment_tree);
```

### 4. Поиск комментариев

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ GET /comments?search=текст&page=1
       ▼
┌─────────────────────────────────────┐
│   Delivery Layer                    │
│   ────────────────────────────────  │
│   1. Парсинг search параметра        │
│   2. Вызов useCase.GetTree()        │
│      (с filter.Search)              │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Use Case Layer                    │
│   ────────────────────────────────  │
│   1. Проверка: filter.Search != "" │
│   2. Вызов repo.Search()            │
│      вместо repo.GetTree()          │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│   Infrastructure Layer              │
│   ────────────────────────────────  │
│   1. SQL с ILIKE для поиска         │
│      WHERE content ILIKE '%текст%'  │
│   2. Рекурсивный запрос для         │
│      получения полного дерева       │
│   3. Возврат []CommentTree           │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│  PostgreSQL │
│  (GIN индекс│
│   для поиска)│
└─────────────┘
```

## Обработка ошибок

### Иерархия ошибок

```
Domain Errors (internal/domain/errors.go)
  ├─ ErrCommentNotFound
  ├─ ErrInvalidParent
  └─ ErrEmptyContent

Use Case Errors
  └─ Оборачивает domain errors с контекстом
     fmt.Errorf("failed to create: %w", domain.ErrEmptyContent)

Delivery Errors
  └─ Преобразует в HTTP статусы:
     domain.ErrCommentNotFound → 404 Not Found
     domain.ErrEmptyContent → 400 Bad Request
     Остальные → 500 Internal Server Error
```

### Пример обработки ошибок

```go
// Use Case
func (uc *CommentUseCase) Create(...) (*domain.Comment, error) {
    if content == "" {
        return nil, domain.ErrEmptyContent  // Доменная ошибка
    }
    
    if err := uc.repo.Create(comment); err != nil {
        return nil, fmt.Errorf("failed to create comment: %w", err)  // Оборачивание
    }
}

// Handler
func (h *CommentHandler) Create(...) {
    comment, err := h.useCase.Create(...)
    if err != nil {
        switch err {
        case domain.ErrEmptyContent:
            http.Error(w, err.Error(), http.StatusBadRequest)  // 400
        case domain.ErrInvalidParent:
            http.Error(w, err.Error(), http.StatusBadRequest)   // 400
        default:
            http.Error(w, "internal server error", http.StatusInternalServerError)  // 500
        }
        return
    }
}
```

## Middleware цепочка

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│ CORSMiddleware      │  ← Добавляет CORS заголовки
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ LoggingMiddleware   │  ← Логирует запрос
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Router              │  ← Маршрутизация
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Handler             │  ← Обработка запроса
└──────────┬──────────┘
           │
           ▼
HTTP Response
```

## Инициализация при старте

```
main.go
  │
  ├─→ config.Load()
  │     └─→ Загрузка из env переменных
  │
  ├─→ slog.New() (JSON handler)
  │     └─→ Инициализация структурированного логирования
  │
  ├─→ pgxpool.New()
  │     └─→ Подключение к PostgreSQL
  │
  ├─→ pool.Ping()
  │     └─→ Проверка соединения с БД
  │
  ├─→ database.NewPostgresRepository(pool)
  │     └─→ Создание репозитория
  │
  ├─→ usecase.NewCommentUseCase(repo)
  │     └─→ Создание use case
  │
  ├─→ http.NewRouter(useCase)
  │     └─→ Создание роутера с handlers
  │
  ├─→ http.FileServer() (для web/)
  │     └─→ Статические файлы веб-интерфейса
  │
  ├─→ CORSMiddleware(router)
  │     └─→ Обертка для CORS
  │
  ├─→ LoggingMiddleware(logger, handler)
  │     └─→ Обертка для логирования
  │
  ├─→ http.Server{} (с таймаутами)
  │     └─→ Настройка сервера (ReadTimeout, WriteTimeout, IdleTimeout)
  │
  ├─→ server.ListenAndServe() (в goroutine)
  │     └─→ Запуск HTTP сервера
  │
  ├─→ signal.Notify() (SIGINT, SIGTERM)
  │     └─→ Ожидание сигнала завершения
  │
  └─→ server.Shutdown() (graceful shutdown)
        └─→ Корректное завершение с таймаутом 30 секунд
```

## Заключение

Все потоки данных следуют принципам Clean Architecture:
- Зависимости направлены внутрь
- Каждый слой решает свою задачу
- Легко тестировать и мокировать
- Легко расширять и изменять

