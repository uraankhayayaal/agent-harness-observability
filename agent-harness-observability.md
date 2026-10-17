# ТЕХНИЧЕСКОЕ ЗАДАНИЕ (PRD & TECHNICAL SPEC)
**Проект:** `agent-harness-observability` (Микросервис и дашборд для анализа траекторий и метрик ИИ-агентов)

---

## 1. Контекст и цель системы
Система предназначена для высокопроизводительного сбора, хронологического анализа и интерактивной визуализации цепочек действий (траекторий) ИИ-агентов, тестируемых в бенчмарках (test harness). Главная цель системы — выявление наиболее эффективных методов работы агентов, подсвечивание аномалий (зацикливаний, пустых мыслей) и расчет стоимости.

### Архитектурный стек:
*   **Брокер сообщений:** Apache Kafka (KRaft режим).
*   **Бэкенд:** Go (Golang) 1.22+, драйверы `pgx/v5` и `neo4j-go-driver/v5`.
*   **Хранилища:** PostgreSQL 16+ (метаданные/агрегаты), Neo4j 5+ (граф траекторий).
*   **Фронтенд:** React (Vite), TailwindCSS, `@xyflow/react` (React Flow).

---

## 2. Архитектура интеграции через Apache Kafka

Харнес асинхронно отправляет логи каждого шага в брокер. Микросервис на Go выступает в роли Consumer Group, вычитывая данные в своем темпе (паттерн CQRS/EDA).

### 2.1. Спецификация топика и продюсера (Инструкция для Харнеса)
*   **Имя топика:** `agent-harness-metrics`
*   **Message Key (Критично):** Строго `run_id` (UUID сессии агента). Это гарантирует привязку всех шагов одного запуска к одной партиции Kafka и сохраняет их строгий хронологический порядок при вычитке, исключая состояние гонки (race condition) в Neo4j.
*   **Message Value (JSON Payload):**
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

### 2.2. Конфигурация Go Consumer
ИИ-разработчик бэкенда должен настроить `kafka.Reader` (`github.com/segmentio/kafka-go`) со следующими параметрами:
*   `GroupID`: `agent-metrics-analyzer`
*   `StartOffset`: `kafka.LastOffset`
*   `CommitInterval`: `0` (коммит оффсета происходит синхронно *только* после успешной записи в обе БД — паттерн *At-Least-Once*).

---

## 3. Спецификация Баз Данных

Микросервис параллельно сохраняет данные в реляционное и графовое хранилища.

### 3.1. PostgreSQL (Реляционный слой)
Используется для хранения метаданных запусков и агрегированных метрик, необходимых для верхнеуровневой аналитики.

```sql
CREATE TABLE test_runs (
    id UUID PRIMARY KEY,
    harness_name VARCHAR(255) NOT NULL,
    agent_version VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'running'
);

CREATE TABLE run_metrics (
    run_id UUID PRIMARY KEY REFERENCES test_runs(id) ON DELETE CASCADE,
    total_steps INT NOT NULL DEFAULT 0,
    total_tokens_used INT NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC(10, 5) NOT NULL DEFAULT 0.0,
    execution_time_ms INT NOT NULL DEFAULT 0
);
```
*Инструкция по UPSERT:* При получении каждого нового шага выполнять `INSERT ... ON CONFLICT (run_id) DO UPDATE` для инкрементирования шагов, токенов, стоимости и времени.

### 3.2. Neo4j (Графовый слой)
Используется для построения направленного графа траектории.
*   **Узлы (Labels):** `Run` (свойство `id`), `Step` (свойства `number`, `thought`, `tool_called`, `status`, `latency_ms`, `tokens_used`).
*   **Связи:** 
    *   `(Run)-[:HAS_FIRST_STEP]->(Step)`
    *   `(Step)-[:NEXT_STEP]->(Step)`

*Cypher-шаблон для воркера бэкенда:*
```cypher
MERGE (r:Run {id: $run_id})
ON CREATE SET r.agent = $agent_version

CREATE (s:Step {
    number: toInteger($step_number),
    thought: $thought,
    tool_called: $tool_called,
    status: $status,
    latency_ms: toInteger($latency_ms),
    tokens_used: toInteger($tokens_used)
})

WITH r, s
CALL {
    WITH r, s
    WITH r, s WHERE $step_number = 1
    MERGE (r)-[:HAS_FIRST_STEP]->(s)
    RETURN count(*) AS c1
}
CALL {
    WITH r, s
    WITH r, s WHERE $step_number > 1
    MATCH (prev:Step {number: toInteger($step_number) - 1})<-[:HAS_FIRST_STEP|NEXT_STEP*]-(r)
    MERGE (prev)-[:NEXT_STEP]->(s)
    RETURN count(*) AS c2
}
RETURN r
```

---

## 4. API Эндпоинт для Фронтенда (Go Бэкенд)

*   **Метод / Путь:** `GET /v1/runs/{run_id}/trajectory`
*   **Логика:** Извлекает путь из Neo4j и форматирует его под структуру React Flow.
*   **Cypher-запрос чтения:**
```cypher
MATCH (r:Run {id: $run_id})-[:HAS_FIRST_STEP]->(f:Step)
MATCH path = (f)-[:NEXT_STEP*0..50]->(m)
WITH nodes(path) AS ns, relationships(path) AS rels
UNWIND ns AS n
UNWIND rels AS rel
RETURN 
  collect(DISTINCT {id: id(n), label: labels(n), props: properties(n)}) AS nodes,
  collect(DISTINCT {id: id(rel), source: id(startNode(rel)), target: id(endNode(rel)), type: type(rel)}) AS edges
```
*   **Выходной формат (JSON):** Массивы `nodes` и `edges` со структурой, совместимой со спецификацией React Flow.

---

## 5. Спецификация Фронтенда (React + React Flow)

Необходимо реализовать SPA-дашборд для визуального дебага траекторий.

### 5.1. Кастомный узел `AgentStepNode`
Компонент, отображающий шаг агента на холсте. Требования к верстке:
*   **Границы (Border):** `border-2 border-blue-500` при успехе; `border-2 border-red-500`, если `status == 'error'`.
*   **Шапка:** Номер шага (слева), `latency_ms` (справа).
*   **Контент:** Иконка и имя `tool_called` (если применимо); текст `thought`, обрезанный до 2 строк с помощью CSS утилиты `line-clamp-2`.
*   **Подвал:** Количество потраченных токенов мелким шрифтом.
*   **Точки подключения:** Два `<Handle>` компонента сверху (`type="target"`) и снизу (`type="source"`).

### 5.2. Экран визуализации (`TrajectoryGraph`)
*   Использовать `<ReactFlow>` с поддержкой свободного зума и панорамирования.
*   Подключить встроенные плагины UI: `<Controls />`, `<MiniMap />`, `<Background variant="dots" />`.
*   Связи (`edges`) делать анимированными (`animated: true`), если шаг привел к успешному выполнению внешнего инструмента.

---

## 6. Инфраструктурное окружение (`docker-compose.yml`)

Для локального запуска всей инфраструктуры ИИ-агент должен использовать данный файл:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: harness-postgres
    environment:
      POSTGRES_USER: harness_user
      POSTGRES_PASSWORD: harness_password
      POSTGRES_DB: harness_db
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U harness_user -d harness_db"]
      interval: 5s
      timeout: 5s
      retries: 5

  neo4j:
    image: neo4j:5.20-community
    container_name: harness-neo4j
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      - NEO4J_AUTH=neo4j/harness_password
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    healthcheck:
      test: ["CMD", "cypher-shell", "-u", "neo4j", "-p", "harness_password", "RETURN 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  kafka:
    image: confluentinc/cp-kafka:7.6.0
    container_name: harness-kafka
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: 'CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT'
      KAFKA_ADVERTISED_LISTENERS: 'PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092'
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REREPLICATION_FACTOR: 1
      KAFKA_PROCESS_ROLES: 'broker,controller'
      KAFKA_LISTENERS: 'PLAINTEXT://0.0.0.0:29092,CONTROLLER://0.0.0.0:29093,PLAINTEXT_HOST://0.0.0.0:9092'
      KAFKA_INTER_BROKER_LISTENER_NAME: 'PLAINTEXT'
      KAFKA_CONTROLLER_LISTENER_NAMES: 'CONTROLLER'
      KAFKA_LOG_DIRS: '/tmp/kraft-combined-logs'
      CLUSTER_ID: 'MkU3OEVBNTcwNTJENDM2Qk'
      KAFKA_CONTROLLER_QUORUM_VOTERS: '1@kafka:29093'
    volumes:
      - kafka_data:/var/lib/kafka/data

  kafdrop:
    image: obsidiandynamics/kafdrop:4.0.1
    container_name: harness-kafdrop
    ports:
      - "9000:9000"
    environment:
      KAFKA_BROKERCONNECT: "kafka:29092"
      JVM_OPTS: "-Xms32M -Xmx64M"
    depends_on:
      - kafka

volumes:
  pgdata:
  neo4j_data:
  neo4j_logs:
  kafka_data:
```

---

## 7. Критерии приемки (Acceptance Criteria для ИИ)
1. **At-Least-Once обработка:** При внезапном падении бэкенда на шаге 3, оффсет в Kafka не должен быть закоммичен. После рестарта бэкенд обязан довычитать и дописать шаг 3.
2. **Идемпотентность графа:** Повторная обработка одного и того же сообщения с одинаковым `run_id` и `step_number` не должна дублировать узлы в Neo4j (использовать `MERGE` или внутренние проверки).
3. **Хронологическая целостность графа:** При сквозном тесте на 10 шагов фронтенд должен отобразить строго линейную цепочку без разрывов в связях `NEXT_STEP`.