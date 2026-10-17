# Инфраструктура (`docker-compose.yml`)

Локальное окружение: PostgreSQL, Neo4j, Kafka (KRaft) и UI для просмотра топиков.

Файл окружения в корне репозитория: [`docker-compose.yml`](../docker-compose.yml).

## Сервисы

| Сервис | Образ | Порты | Назначение |
|---|---|---|---|
| `postgres` | `postgres:16-alpine` | `5432` | Метаданные и агрегаты (`harness_db`) |
| `neo4j` | `neo4j:5.20-community` | `7474` (HTTP), `7687` (Bolt) | Граф траекторий |
| `kafka` | `confluentinc/cp-kafka:7.6.0` | `9092` | Брокер сообщений (KRaft, broker+controller) |
| `kafdrop` | `obsidiandynamics/kafdrop:4.0.1` | `9000` | Web UI для просмотра Kafka |

## Учетные данные (локальные)

| Сервис | Пользователь | Пароль | База |
|---|---|---|---|
| PostgreSQL | `harness_user` | `harness_password` | `harness_db` |
| Neo4j | `neo4j` | `harness_password` | — |

## Состав docker-compose

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

## Запуск

```bash
docker-compose up -d
```

## Проверка

- PostgreSQL: `pg_isready -U harness_user -d harness_db -h localhost -p 5432`
- Neo4j HTTP UI: `http://localhost:7474`
- Kafka: `kafka-topics --bootstrap-server localhost:9092 --list`
- Kafdrop: `http://localhost:9000`

Связанные документы:
- [Интеграция через Kafka](./integration.md)
- [Базы данных](./database.md)
