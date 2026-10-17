package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr    string
	APIBasePath string

	KafkaBrokers             []string
	KafkaTopic               string
	KafkaGroupID             string
	KafkaStartOffset         string
	KafkaFetchMinBytes       int
	KafkaFetchMaxBytes       int
	KafkaMaxWait             time.Duration
	KafkaSessionTimeout      time.Duration
	KafkaHeartbeatInterval   time.Duration
	KafkaAcks                string
	KafkaCompression         string
	KafkaBatchSize           int
	KafkaLinger              time.Duration

	PostgresURL string

	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
}

func Load() Config {
	return Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		APIBasePath: normalizeBasePath(getEnv("API_BASE_PATH", "/v1")),

		KafkaBrokers:           splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:             getEnv("KAFKA_TOPIC", "agent-harness-metrics"),
		KafkaGroupID:           getEnv("KAFKA_GROUP_ID", "agent-metrics-analyzer"),
		KafkaStartOffset:       getEnv("KAFKA_START_OFFSET", "last"),
		KafkaFetchMinBytes:     getEnvInt("KAFKA_FETCH_MIN_BYTES", 1024),
		KafkaFetchMaxBytes:     getEnvInt("KAFKA_FETCH_MAX_BYTES", 52428800),
		KafkaMaxWait:           getEnvMS("KAFKA_MAX_WAIT_MS", 500),
		KafkaSessionTimeout:    getEnvMS("KAFKA_SESSION_TIMEOUT_MS", 30000),
		KafkaHeartbeatInterval: getEnvMS("KAFKA_HEARTBEAT_INTERVAL_MS", 10000),
		KafkaAcks:              getEnv("KAFKA_ACKS", "all"),
		KafkaCompression:       getEnv("KAFKA_COMPRESSION", "snappy"),
		KafkaBatchSize:         getEnvInt("KAFKA_BATCH_SIZE", 16384),
		KafkaLinger:            getEnvMS("KAFKA_LINGER_MS", 10),

		PostgresURL: getEnv("POSTGRES_URL", "postgres://harness_user:harness_password@localhost:5432/harness_db?sslmode=disable"),

		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "harness_password"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvMS(key string, fallbackMS int) time.Duration {
	return time.Duration(getEnvInt(key, fallbackMS)) * time.Millisecond
}

func normalizeBasePath(p string) string {
	if p == "" {
		return "/v1"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
