# Диаграммы архитектуры CommentTree

## Общая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                        Client                               │
│                    (Browser/API Client)                      │
└───────────────────────────┬─────────────────────────────────┘
                            │ HTTP
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Delivery Layer                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HTTP Router                                          │  │
│  │  - POST /comments                                     │  │
│  │  - GET /comments                                      │  │
│  │  - DELETE /comments/{id}                             │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Middleware                                           │  │
│  │  - CORS                                               │  │
│  │  - Logging                                            │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Handlers                                             │  │
│  │  - Create()                                           │  │
│  │  - GetTree()                                          │  │
│  │  - Delete()                                           │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Use Case Layer                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  CommentUseCase                                       │  │
│  │  - Create(ctx, parentID, content)                     │  │
│  │  - GetTree(ctx, filter)                               │  │
│  │  - Delete(ctx, id)                                    │  │
│  │  - GetTotalCount(ctx, parentID, search)               │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                        Domain Layer                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Models                                                │  │
│  │  - Comment                                             │  │
│  │  - CommentTree                                         │  │
│  │  - CommentFilter                                       │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Interfaces                                           │  │
│  │  CommentRepository {                                  │  │
│  │    Create()                                           │  │
│  │    GetByID()                                          │  │
│  │    GetTree()                                          │  │
│  │    Delete()                                           │  │
│  │    Search()                                           │  │
│  │    Count()                                            │  │
│  │  }                                                     │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Errors                                                │  │
│  │  - ErrCommentNotFound                                  │  │
│  │  - ErrInvalidParent                                   │  │
│  │  - ErrEmptyContent                                    │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Infrastructure Layer                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  PostgresRepository                                    │  │
│  │  (implements CommentRepository)                        │  │
│  │  - Рекурсивные SQL запросы                             │  │
│  │  - Построение дерева                                   │  │
│  │  - Полнотекстовый поиск                                │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      PostgreSQL                               │
│  - Таблица comments                                           │
│  - Индексы для поиска                                        │
│  - Рекурсивные запросы                                       │
└─────────────────────────────────────────────────────────────┘
```

## Поток создания комментария

```
┌─────────┐
│ Client  │
└────┬────┘
     │ POST /comments
     │ {"parent_id": 1, "content": "Текст"}
     ▼
┌─────────────────────────────────────┐
│ Handler.Create()                     │
│ 1. Парсинг JSON                      │
│ 2. Валидация структуры                │
│ 3. useCase.Create(...)               │
└────┬─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ UseCase.Create()                     │
│ 1. Проверка: content != ""          │
│ 2. Если parentID != nil:             │
│    repo.GetByID(parentID)           │
│ 3. repo.Create(comment)             │
└────┬─────────────────────────────────┘
     │
     ├─────────────────┐
     │                 │
     ▼                 ▼
┌──────────┐    ┌──────────────┐
│ Repo.    │    │ Repo.        │
│ GetByID()│    │ Create()     │
└────┬──────┘    └──────┬───────┘
     │                  │
     └────────┬─────────┘
              │
              ▼
     ┌──────────────┐
     │ PostgreSQL   │
     │ SELECT ...   │
     │ INSERT ...   │
     └──────────────┘
```

## Поток получения дерева

```
┌─────────┐
│ Client  │
└────┬────┘
     │ GET /comments?page=1&page_size=20
     ▼
┌─────────────────────────────────────┐
│ Handler.GetTree()                    │
│ 1. Парсинг query параметров         │
│ 2. Создание CommentFilter           │
│ 3. useCase.GetTree(filter)          │
│ 4. useCase.GetTotalCount()          │
│ 5. Преобразование в DTO             │
└────┬─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ UseCase.GetTree()                   │
│ 1. Нормализация параметров          │
│    - page, pageSize, sortBy, order  │
│ 2. Если search != "":               │
│    repo.Search()                   │
│    Иначе:                          │
│    repo.GetTree()                  │
└────┬─────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ Repo.GetTree()                       │
│ 1. WITH RECURSIVE comment_tree AS    │
│    (SELECT ... UNION ALL ...)       │
│ 2. Получение всех комментариев      │
│ 3. Построение дерева (buildTree)    │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────┐
│ PostgreSQL   │
│ Рекурсивный  │
│ SQL запрос   │
└──────────────┘
```

## Иерархия зависимостей

```
                    ┌─────────────┐
                    │   main.go   │
                    │ (инициализация)│
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                   │
        ▼                  ▼                   ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   config      │  │ delivery     │  │ infrastructure│
│               │  │              │  │              │
│ Load()        │  │ Handler      │  │ PostgresRepo │
└───────────────┘  │ Router       │  └──────┬───────┘
                   │ Middleware   │         │
                   └──────┬───────┘         │
                          │                 │
                          ▼                 │
                   ┌──────────────┐         │
                   │   usecase    │         │
                   │              │         │
                   │ CommentUseCase│       │
                   └──────┬───────┘         │
                          │                 │
                          ▼                 │
                   ┌──────────────┐         │
                   │    domain    │◄────────┘
                   │              │
                   │ Comment      │
                   │ CommentTree  │
                   │ CommentRepository│
                   │ (interface)   │
                   └──────────────┘
```

**Принцип**: Все зависимости направлены внутрь к domain слою.

## Структура данных

### Comment (Доменная модель)
```
┌─────────────────────────┐
│      Comment            │
├─────────────────────────┤
│ ID: int64               │
│ ParentID: *int64        │  ← nil для корневых
│ Content: string         │
│ CreatedAt: time.Time    │
│ UpdatedAt: time.Time    │
└─────────────────────────┘
```

### CommentTree (Дерево)
```
┌─────────────────────────┐
│    CommentTree          │
├─────────────────────────┤
│ Comment: Comment        │
│ Children: []CommentTree │  ← Рекурсивная структура
└─────────────────────────┘
         │
         ├─── Comment (ID: 1)
         │     └─── CommentTree (ID: 2, parent: 1)
         │           └─── CommentTree (ID: 3, parent: 2)
         │
         └─── Comment (ID: 4)
               └─── CommentTree (ID: 5, parent: 4)
```

### CommentFilter (Фильтры)
```
┌─────────────────────────┐
│   CommentFilter         │
├─────────────────────────┤
│ ParentID: *int64        │  ← Фильтр по родителю
│ Search: string          │  ← Поисковый запрос
│ Page: int               │  ← Номер страницы
│ PageSize: int           │  ← Размер страницы
│ SortBy: string          │  ← Поле сортировки
│ Order: string           │  ← Порядок (asc/desc)
└─────────────────────────┘
```

## Схема БД

```
┌─────────────────────────────┐
│        comments              │
├─────────────────────────────┤
│ id: BIGSERIAL PRIMARY KEY   │
│ parent_id: BIGINT           │  ← FK to comments(id)
│ content: TEXT               │
│ created_at: TIMESTAMP       │
│ updated_at: TIMESTAMP       │
└─────────────────────────────┘
         │
         │ ON DELETE CASCADE
         │
         ▼
    (self-reference)
```

**Индексы**:
- `idx_comments_parent_id` - для быстрого поиска дочерних
- `idx_comments_created_at` - для сортировки
- `idx_comments_content_trgm` - GIN индекс для полнотекстового поиска

## Middleware цепочка

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│ CORSMiddleware      │  ← Добавляет CORS заголовки
│                     │     Access-Control-Allow-*
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ LoggingMiddleware   │  ← Логирует запрос
│                     │     method, path, duration
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Router              │  ← Маршрутизация
│                     │     POST /comments
│                     │     GET /comments
│                     │     DELETE /comments/{id}
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Handler             │  ← Обработка запроса
│                     │     Парсинг, валидация
│                     │     Вызов use case
└──────────┬──────────┘
           │
           ▼
HTTP Response
```

## Инициализация приложения

```
main()
  │
  ├─→ config.Load()
  │     └─→ Чтение env переменных (приоритет: env > .env > defaults)
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
  │     └─→ Создание роутера
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
  │     └─→ ReadTimeout: 15s, WriteTimeout: 15s, IdleTimeout: 60s
  │
  ├─→ server.ListenAndServe() (в goroutine)
  │     └─→ Запуск HTTP сервера
  │
  ├─→ signal.Notify() (SIGINT, SIGTERM)
  │     └─→ Ожидание сигнала завершения
  │
  └─→ server.Shutdown() (graceful shutdown)
        └─→ Таймаут: 30 секунд, корректное закрытие соединений
```

## Обработка ошибок

```
Domain Layer
  ├─→ ErrCommentNotFound
  ├─→ ErrInvalidParent
  └─→ ErrEmptyContent
        │
        ▼
Use Case Layer
  └─→ Оборачивание с контекстом
      fmt.Errorf("failed to create: %w", domain.ErrEmptyContent)
        │
        ▼
Delivery Layer
  └─→ Преобразование в HTTP статусы
      ErrCommentNotFound → 404 Not Found
      ErrEmptyContent → 400 Bad Request
      Остальные → 500 Internal Server Error
```

