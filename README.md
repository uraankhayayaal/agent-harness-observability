# Agent Harness Observability

Микросервис и дашборд для анализа траекторий и метрик ИИ-агентов: сбор логов шагов через Apache Kafka, хранение в PostgreSQL и Neo4j, визуализация траекторий на React Flow.

## Документация

- [Обзор проекта](./docs/overview.md) — цель, стек, поток данных
- [Архитектура](./docs/architecture.md) — общий дизайн, CQRS/EDA
- [Интеграция через Kafka](./docs/integration.md) — топик, формат сообщений, настройки producer/consumer
- [Базы данных](./docs/database.md) — схемы PostgreSQL и Neo4j
- [API](./docs/api.md) — эндпоинты для фронтенда
- [Фронтенд](./docs/frontend.md) — спецификация дашборда
- [Инфраструктура](./docs/infrastructure.md) — docker-compose окружение
- [Критерии приемки](./docs/acceptance.md) — условия готовности
- [Правила работы агентов (AGENTS.md)](./AGENTS.md)

## Структура проекта

- `backend/` — серверная часть (Go)
- `frontend/` — клиентская часть (React)
- `tests/` — e2e-тесты (Playwright): полный прогон Kafka → backend → API → UI
- `docs/` — документация проекта
- `docker-compose.yml` — инфраструктура (PostgreSQL, Neo4j, Kafka, Kafdrop) + приложения (backend, frontend)

## Быстрый старт

### Всё в Docker (рекомендуется)

1. Создать локальный файл окружения: `cp .env.example .env`
2. Поднять весь стек: `make up` — инфраструктура + backend (Go) + frontend (nginx)
   - UI: http://localhost:8081 (порт задаётся `WEB_PORT`), API: http://localhost:8080
3. Отправить демо-сообщения в Kafka: `make up-producer`
4. Остановить: `make down`

Dev-режим с hot-reload (backend — `go run` с mount исходников, frontend — vite): `make up-dev` (UI на http://localhost:5173).

### Локально на хосте

1. Создать локальный файл окружения: `cp .env.example .env` (все глобальные константы: порты, эндпоинты, топики, креды, настройки Kafka)
2. Запустить инфраструктуру: `make infra-up` (см. [Инфраструктуру](./docs/infrastructure.md))
3. Запустить бэкенд/producer/фронтенд: `make backend`, `make producer`, `make frontend`
4. Интеграция с харнесом через Kafka: топик и настройки описаны в [Интеграции](./docs/integration.md) и задаются в `.env`

## Тесты

- Юнит-тесты бэкенда: `make test`
- E2E (полный прогон: сообщение в Kafka → бэкенд → API → визуализация во фронтенде):
  ```bash
  make infra-up        # инфраструктура (Kafka, PostgreSQL, Neo4j)
  make e2e-install     # однократная установка Playwright + Chromium
  make e2e             # прогон (backend/frontend поднимутся автоматически)
  ```
  Скриншот визуализации сохраняется в `tests/test-results/`.

## CI/CD

GitHub Actions (`.github/workflows/docker.yml`) собирает образы `backend` и `frontend`:
на pull request — только сборка (проверка), на push в `main` — сборка и публикация в GHCR с тегами `latest` и `sha-<short>`:

```
ghcr.io/uraankhayayaal/agent-harness-observability/backend:latest
ghcr.io/uraankhayayaal/agent-harness-observability/frontend:latest
```
