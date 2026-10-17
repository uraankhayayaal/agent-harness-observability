# Базы данных

Микросервис параллельно сохраняет данные в реляционное и графовое хранилища.

## 1. PostgreSQL (реляционный слой)

Хранит метаданные запусков и агрегированные метрики, необходимые для верхнеуровневой аналитики.

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

**Инструкция по UPSERT:** при получении каждого нового шага выполнять `INSERT ... ON CONFLICT (run_id) DO UPDATE` для инкрементирования шагов, токенов, стоимости и времени.

```sql
INSERT INTO run_metrics (run_id, total_steps, total_tokens_used, total_cost_usd, execution_time_ms)
VALUES ($1, 1, $2, $3, $4)
ON CONFLICT (run_id) DO UPDATE SET
    total_steps        = run_metrics.total_steps + 1,
    total_tokens_used  = run_metrics.total_tokens_used + EXCLUDED.total_tokens_used,
    total_cost_usd     = run_metrics.total_cost_usd + EXCLUDED.total_cost_usd,
    execution_time_ms  = run_metrics.execution_time_ms + EXCLUDED.execution_time_ms;
```

## 2. Neo4j (графовый слой)

Строит направленный граф траектории.

### Узлы (Labels)

| Label | Свойства |
|---|---|
| `Run` | `id` |
| `Step` | `number`, `thought`, `tool_called`, `status`, `latency_ms`, `tokens_used` |

### Связи

- `(Run)-[:HAS_FIRST_STEP]->(Step)`
- `(Step)-[:NEXT_STEP]->(Step)`

### Cypher-шаблон записи (для воркера бэкенда)

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

**Идемпотентность:** повторная обработка одного и того же сообщения с одинаковым `run_id` и `step_number` не должна дублировать узлы в Neo4j — использовать `MERGE` или внутренние проверки (см. [Критерии приемки](./acceptance.md)).

Связанные документы:
- [Интеграция через Kafka](./integration.md)
- [Инфраструктура (запуск БД)](./infrastructure.md)
