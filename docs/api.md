# API (Go-бэкенд)

REST-эндпоинты для фронтенда.

## `GET /v1/runs/{run_id}/trajectory`

**Логика:** извлекает путь траектории из Neo4j и форматирует его под структуру React Flow.

### Параметры

| Параметр | Тип | Описание |
|---|---|---|
| `run_id` (path) | UUID | Идентификатор запуска агента |

### Cypher-запрос чтения

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

### Выходной формат (JSON)

Массивы `nodes` и `edges` со структурой, совместимой со спецификацией React Flow:

```json
{
  "nodes": [
    {
      "id": "<internal id>",
      "label": ["Run" | "Step"],
      "props": {
        "number": 1,
        "thought": "...",
        "tool_called": "...",
        "status": "success",
        "latency_ms": 120,
        "tokens_used": 42
      }
    }
  ],
  "edges": [
    {
      "id": "<internal id>",
      "source": "<source node id>",
      "target": "<target node id>",
      "type": "NEXT_STEP"
    }
  ]
}
```

Связанные документы:
- [Базы данных](./database.md) — модель `Run`/`Step`
- [Фронтенд](./frontend.md) — отрисовка узлов и связей
