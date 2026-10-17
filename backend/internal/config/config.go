package config

import (
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
	PostgresURL   string
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
	HTTPAddr      string
}

func Load() Config {
	return Config{
		KafkaBrokers:  splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:    getEnv("KAFKA_TOPIC", "agent-harness-metrics"),
		KafkaGroupID:  getEnv("KAFKA_GROUP_ID", "agent-metrics-analyzer"),
		PostgresURL:   getEnv("POSTGRES_URL", "postgres://harness_user:harness_password@localhost:5432/harness_db?sslmode=disable"),
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "harness_password"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
