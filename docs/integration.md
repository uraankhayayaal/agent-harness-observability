# Интеграция через Kafka

Интеграция с харнесом — **асинхронная, через Apache Kafka** (KRaft). Харнес публикует события шагов в топик, Go-микросервис потребляет их через Consumer Group (паттерн CQRS/EDA).

## 1. Топик

| Параметр | Значение |
|---|---|
| **Имя топика** | `agent-harness-metrics` |
| **Назначение** | Логи каждого шага выполнения агента (траектория + метрики) |
| **Формат значения** | JSON (UTF-8) |

> Топик создавать заранее (декларативно), не полагаться на авто-создание в production.

## 2. Message Key (критично)

Строго **`run_id`** (UUID сессии агента).

Это гарантирует привязку всех шагов одного запуска к одной партиции Kafka и сохраняет их строгий хронологический порядок при вычитке, исключая состояние гонки (race condition) в Neo4j.

```bash
# Эквивалент при отправке
--property parse.key=true --property key.separator='|' --property key=<run_id>
```

## 3. Формат сообщения (Message Value, JSON)

```json
{
  "run_id": "string (uuid)",
  "harness_name": "string",
  "agent_version": "string",
  "step_number": "integer",
  "thought": "string",
  "tool_called": "string (optional)",
  "status": "string ('success' | 'error')",
  "latency_ms": "integer",
  "tokens_used": "integer",
  "cost_usd": "float"
}
```

**Поля:**

| Поле | Тип | Обязательное | Описание |
|---|---|---|---|
| `run_id` | string (uuid) | Да | Идентификатор запуска агента — **ключ сообщения** |
| `harness_name` | string | Да | Имя харнеса/бенчмарка |
| `agent_version` | string | Да | Версия тестируемого агента |
| `step_number` | integer | Да | Порядковый номер шага (с 1) |
| `thought` | string | Да | Текст рассуждения агента на шаге |
| `tool_called` | string | Нет | Имя вызванного инструмента (если был вызов) |
| `status` | string | Да | `success` или `error` |
| `latency_ms` | integer | Да | Латентность шага, мс |
| `tokens_used` | integer | Да | Израсходованные токены |
| `cost_usd` | float | Да | Стоимость шага в USD |

## 4. Настройки Consumer (Go)

Конфигурация `kafka.Reader` (`github.com/segmentio/kafka-go`):

| Параметр | Значение | Пояснение |
|---|---|---|
| `GroupID` | `agent-metrics-analyzer` | Consumer Group ID. Все инстансы бэкенда в одной группе делят партиции |
| `StartOffset` | `kafka.LastOffset` | Читать с последнего смещения (события до старта консюмера не обрабатываются) |
| `CommitInterval` | `0` | Коммит оффсета происходит **синхронно только после успешной записи в обе БД** — паттерн *At-Least-Once* |

Пример:

```go
r := kafka.NewReader(kafka.ReaderConfig{
    Brokers:        strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
    Topic:          "agent-harness-metrics",
    GroupID:        "agent-metrics-analyzer",
    StartOffset:    kafka.LastOffset,
    CommitInterval: 0,
})
```

## 5. Настройки Producer (харнес)

| Параметр | Рекомендуемое значение | Пояснение |
|---|---|---|
| `bootstrap.servers` | адреса брокера (например `localhost:9092`) | Broker list |
| `message key` | `run_id` | **Обязательно** — порядок шагов запуска |
| `acks` | `all` | Гарантия доставки (at-least-once) |
| `retries` | `10` | Ретраи при временных ошибках |
| `retry.backoff.ms` | `500` | Задержка между ретраями |
| `enable.idempotence` | `true` | Защита от дублей при ретраях |
| `compression.type` | `snappy` (или `lz4`, `zstd`) | Компрессия батча |
| `batch.size` | `16384` (16 KB) | Максимальный размер батча, байты |
| `linger.ms` | `5–20` | Задержка ожидания заполнения батча (батчинг: растит пропускную способность ценой небольшой латентности) |
| `buffer.memory` | `33554432` (32 MB) | Буфер неподтвержденных сообщений |

## 6. Дополнительные настройки Consumer

| Параметр | Рекомендуемое значение | Пояснение |
|---|---|---|
| `auto.offset.reset` | `latest` (соответствует `StartOffset: LastOffset`) | Поведение при отсутствии сохраненного оффсета |
| `enable.auto.commit` | `false` (эквивалент `CommitInterval: 0`) | Только ручной коммит после записи в БД |
| `max.poll.records` | `100–500` | Батч чтения: сколько сообщений за один `poll()` |
| `fetch.min.bytes` | `1024` | Минимум байт перед возвратом с брокера (накопление батча) |
| `fetch.max.bytes` | `52428800` (50 MB) | Максимум за fetch |
| `max.partition.fetch.bytes` | `1048576` (1 MB) | Максимум с одной партиции |
| `session.timeout.ms` | `30000` (30s) | Таймаут сессии |
| `heartbeat.interval.ms` | `10000` (10s) | Интервал heartbeats (~1/3 session timeout) |
| `isolation.level` | `read_committed` | Чтение подтвержденных транзакций |
| `max.poll.interval.ms` | `300000` (5 min) | Лимит времени обработки батча между poll (увеличивать, если запись в обе БД долгая) |

## 7. Обработка ошибок

| Сценарий | Стратегия |
|---|---|
| Временные ошибки (таймаут, reconnect) | Ретраи с backoff |
| Ошибка записи в БД | **Не коммитить оффсет**, вернуться к сообщению после рестарта (At-Least-Once) |
| Битое/невалидное сообщение | Dead Letter Queue: топик `agent-harness-metrics.dlq` + заголовки `error.reason`, `retry.count` |

## 8. Переменные окружения (`.env`)

Все глобальные константы (эндпоинты, порты, топики, креды к БД, настройки Kafka) вынесены в файлы окружения в корне проекта:

- `.env.example` — шаблон, коммитится в репозиторий
- `.env` — локальные значения (создать: `cp .env.example .env`; в `.gitignore`)

Makefile загружает `.env` автоматически (задачи `infra-up`, `backend`, `producer`, `frontend`); docker compose подхватывает `.env` сам; Vite читает `VITE_*` переменные из корневого `.env`.

### HTTP / API

| Переменная | Дефолт | Описание |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера бэкенда |
| `API_BASE_PATH` | `/v1` | Базовый путь API (`{API_BASE_PATH}/runs/{run_id}/trajectory`) |
| `LOG_LEVEL` | `info` | Уровень логирования (`debug`/`info`) |

### Kafka

| Переменная | Дефолт | Описание |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Список брокеров через запятую |
| `KAFKA_TOPIC` | `agent-harness-metrics` | Топик событий шагов |
| `KAFKA_GROUP_ID` | `agent-metrics-analyzer` | Consumer Group ID |
| `KAFKA_START_OFFSET` | `last` | Начальное смещение (`last`/`first`) |
| `KAFKA_FETCH_MIN_BYTES` | `1024` | Минимум байт для fetch (батчинг чтения) |
| `KAFKA_FETCH_MAX_BYTES` | `52428800` | Максимум байт за fetch |
| `KAFKA_MAX_WAIT_MS` | `500` | Максимальное ожидание батча на брокере |
| `KAFKA_SESSION_TIMEOUT_MS` | `30000` | Session timeout |
| `KAFKA_HEARTBEAT_INTERVAL_MS` | `10000` | Heartbeat interval |
| `KAFKA_ACKS` | `all` | acks producer'а (`all`/`one`/`none`) |
| `KAFKA_COMPRESSION` | `snappy` | Компрессия producer'а (`snappy`/`lz4`/`zstd`/`gzip`/`none`) |
| `KAFKA_BATCH_SIZE` | `16384` | Размер батча producer'а, байты |
| `KAFKA_LINGER_MS` | `10` | Linger producer'а (ожидание заполнения батча) |

### PostgreSQL

| Переменная | Дефолт | Описание |
|---|---|---|
| `POSTGRES_URL` | `postgres://harness_user:harness_password@localhost:5432/harness_db?sslmode=disable` | DSN бэкенда |
| `POSTGRES_USER` | `harness_user` | Пользователь (docker-compose) |
| `POSTGRES_PASSWORD` | `harness_password` | Пароль (docker-compose) |
| `POSTGRES_DB` | `harness_db` | Имя базы (docker-compose) |
| `POSTGRES_PORT` | `5432` | Порт (docker-compose) |

### Neo4j

| Переменная | Дефолт | Описание |
|---|---|---|
| `NEO4J_URI` | `bolt://localhost:7687` | Bolt URI бэкенда |
| `NEO4J_USER` | `neo4j` | Пользователь |
| `NEO4J_PASSWORD` | `harness_password` | Пароль |
| `NEO4J_HTTP_PORT` | `7474` | HTTP-порт (docker-compose) |
| `NEO4J_BOLT_PORT` | `7687` | Bolt-порт (docker-compose) |

### Frontend (Vite)

| Переменная | Дефолт | Описание |
|---|---|---|
| `FRONTEND_PORT` | `5173` | Порт dev-сервера Vite |
| `VITE_API_PROXY_TARGET` | `http://localhost:8080` | Цель прокси dev-сервера |
| `VITE_API_BASE_PATH` | `/v1` | Базовый путь API на клиенте (должен совпадать с `API_BASE_PATH`) |
| `KAFKA_PORT` | `9092` | Порт Kafka broker (docker-compose) |
| `KAFDROP_PORT` | `9000` | Порт Kafdrop (docker-compose) |

## 9. Проверка интеграции

```bash
# Список топиков
kafka-topics --bootstrap-server localhost:9092 --list

# Описание топика (партиции, key)
kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic agent-harness-metrics

# Отправка тестового сообщения с ключом run_id
echo '{"run_id":"3f6a2b1e-1111-4a11-8a11-000000000001","harness_name":"test","agent_version":"1.0","step_number":1,"thought":"test thought","tool_called":"bash","status":"success","latency_ms":120,"tokens_used":42,"cost_usd":0.001}' | \
kafka-console-producer --bootstrap-server localhost:9092 \
  --topic agent-harness-metrics \
  --property parse.key=true --property key.separator='|' \
  --property key=3f6a2b1e-1111-4a11-8a11-000000000001

# Чтение из топика
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic agent-harness-metrics \
  --group agent-metrics-analyzer \
  --property print.key=true --property key.separator='|'
```

UI-просмотр: Kafdrop доступен на `http://localhost:9000` (см. [Инфраструктуру](./infrastructure.md)).

## 10. Рекомендации

- **Партиционирование:** всегда по `run_id` — сохраняет порядок шагов одного запуска.
- **Батчинг:** использовать `linger.ms` + `batch.size` на продюсере и `max.poll.records` на консюмере.
- **Модель доставки:** **At-Least-Once** — коммит оффсета только после успешной записи в PostgreSQL и Neo4j (см. [Критерии приемки](./acceptance.md)).
- **Идемпотентность:** повторная обработка сообщения не должна дублировать данные (см. [Базы данных](./database.md)).
