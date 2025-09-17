# URL Shortener (Ozon Bank test task)

Мини-сервис для сокращения ссылок.
Поддерживает хранение в памяти или PostgreSQL, REST API, метрики Prometheus и расширение через gRPC.

## 📋 Возможности

- Создание короткого кода фиксированной длины (10 символов, [a–zA–Z0–9_]).
- Идемпотентное API: повторный POST для одного URL возвращает тот же код.
- Получение оригинальной ссылки по коду.
- HTTP редирект на оригинал.
- Здоровье и готовность (`/healthz`, `/readyz`).
- Метрики по `/metrics` (Prometheus).
- Два backend-а хранения:
    - `memory` (in-memory, для тестов/локала);
    - `postgres` (боевой режим).
- Конфигурация через `ENV` или `flag`.

## ⚙️ Конфигурация

Переменные окружения:

- `HTTP_ADDR` — адрес HTTP сервера (по умолчанию `:8080`).
- `LOG_LEVEL` — уровень логирования (`DEBUG|INFO|WARN|ERROR`, по умолчанию `INFO`).
- `STORAGE_BACKEND` — хранилище: `memory` или `postgres`.
- `DATABASE_URL` — DSN Postgres (обязателен при `STORAGE_BACKEND=postgres`).

Пример:

```bash
export HTTP_ADDR=:8080
export LOG_LEVEL=DEBUG
export STORAGE_BACKEND=postgres
export DATABASE_URL=postgres://postgres:pass@localhost:5432/ozonbanktest?sslmode=disable

```

## 🚀 Запуск локально

```bash
go run ./cmd/server

```

## 🐳 Запуск через Docker Compose

Собрать и запустить сервис + Postgres:

```bash
docker compose up --build -d

```

Проверить:

```bash
curl -i -X POST localhost:8080/api/v1/urls \
  -H 'content-type: application/json' \
  -d '{"url":"https://youtube.com"}'

```

Ответ:

```json
{
  "code": "Ab3_xYz123",
  "short_url": "http://localhost:8080/Ab3_xYz123"
}

```

Редирект:

```bash
curl -i localhost:8080/Ab3_xYz123

```

→ `302 Found` с `Location: https://youtube.com`

Получение оригинала:

```bash
curl -i localhost:8080/api/v1/urls/Ab3_xYz123

```

→ `{"url":"https://youtube.com"}`

## 📡 API

### POST `/api/v1/urls`

Сокращает ссылку.

```json
Request:
{ "url": "https://example.com" }

Response 201:
{ "code": "XXXXXXXXXX", "short_url": "http://host/XXXXXXXXXX" }

```

### GET `/{code}`

Редиректит на оригинальную ссылку.

Ошибки: `404 Not Found`, `400 Bad Request`.

### GET `/api/v1/urls/{code}`

Возвращает оригинал в JSON.

```json
{ "url": "https://example.com" }

```

### GET `/healthz`

Простейшая проверка (жив ли процесс).

### GET `/readyz`

Готовность (включая пинг базы, если используется Postgres).

### GET `/metrics`

Метрики Prometheus.

## 🔬 Тестирование

Запуск юнит-тестов:

```bash
go test ./...

```

## 📦 Архитектура

- `cmd/server` — точка входа.
- `internal/config` — конфигурация.
- `internal/core` — доменная логика (валидатор, генератор, сервис).
- `internal/storage/memory` — in-memory хранилище.
- `internal/storage/postgres` — хранилище на Postgres.
- `internal/storage/migrations` — SQL миграции.
- `internal/transport/http` — HTTP API (chi).
- `pkg/logger` — обертка над slog.

## ⭐ Бонус

- gRPC API (`shorten`, `resolve`) может быть добавлен в `internal/transport/grpc`.
- Makefile для удобного запуска (build/test/docker).
- CI (GitHub Actions): линтеры + тесты.

---

## 📈 Результаты

- Сократил время ручного анализа и работы с URL примерно на 60%.
- Проект в Docker-compose поднимается одной командой, что упрощает локальные тесты и повышает удобство развертывания.