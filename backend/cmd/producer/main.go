package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func requiredAcks(s string) kafka.RequiredAcks {
	switch s {
	case "none":
		return kafka.RequireNone
	case "one":
		return kafka.RequireOne
	default:
		return kafka.RequireAll
	}
}

func compression(s string) kafka.Compression {
	switch s {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return 0
	}
}

type demoTool struct {
	name    string
	thought string
	status  domain.StepStatus
}

func main() {
	brokers := flag.String("brokers", envOr("KAFKA_BROKERS", "localhost:9092"), "comma-separated kafka brokers")
	topic := flag.String("topic", envOr("KAFKA_TOPIC", "agent-harness-metrics"), "kafka topic")
	runID := flag.String("run-id", "", "run id (uuid), generated if empty")
	steps := flag.Int("steps", 10, "number of steps to produce")
	flag.Parse()

	id := *runID
	if id == "" {
		id = newUUID()
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(*brokers, ",")...),
		Topic:        *topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: requiredAcks(envOr("KAFKA_ACKS", "all")),
		Compression:  compression(envOr("KAFKA_COMPRESSION", "snappy")),
		BatchSize:    envOrInt("KAFKA_BATCH_SIZE", 16384),
		BatchTimeout: time.Duration(envOrInt("KAFKA_LINGER_MS", 10)) * time.Millisecond,
	}
	defer writer.Close()

	ctx := context.Background()
	for i := 1; i <= *steps; i++ {
		step := demoStep(id, i)
		payload, err := json.Marshal(step)
		if err != nil {
			log.Fatal(err)
		}
		// Key = run_id keeps all steps of a run in one partition (strict order).
		if err := writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(id),
			Value: payload,
		}); err != nil {
			log.Fatalf("write message: %v", err)
		}
	}

	fmt.Printf("produced %d steps for run_id=%s\n", *steps, id)
}

func demoStep(runID string, n int) domain.AgentStep {
	tools := []demoTool{
		{name: "", thought: "I need to inspect the repository structure first.", status: domain.StepStatusSuccess},
		{name: "bash", thought: "Listing files in the working directory.", status: domain.StepStatusSuccess},
		{name: "read_file", thought: "Reading the main entry point to understand the wiring.", status: domain.StepStatusSuccess},
		{name: "search", thought: "Searching for all references of the config loader.", status: domain.StepStatusSuccess},
		{name: "read_file", thought: "Opening the config module found by search.", status: domain.StepStatusSuccess},
		{name: "edit_file", thought: "Applying the fix to the config validation branch.", status: domain.StepStatusSuccess},
		{name: "bash", thought: "Running unit tests to verify the change.", status: domain.StepStatusSuccess},
		{name: "bash", thought: "Tests failed due to a flaky fixture, re-running the suite.", status: domain.StepStatusError},
		{name: "bash", thought: "All tests are green now.", status: domain.StepStatusSuccess},
		{name: "", thought: "Change verified, summarizing the result for the user.", status: domain.StepStatusSuccess},
	}
	t := tools[(n-1)%len(tools)]
	return domain.AgentStep{
		RunID:        runID,
		HarnessName:  "swe-bench-lite",
		AgentVersion: "demo-1.0.0",
		StepNumber:   n,
		Thought:      t.thought,
		ToolCalled:   t.name,
		Status:       t.status,
		LatencyMs:    100 + n*37,
		TokensUsed:   250 + n*15,
		CostUSD:      float64(250+n*15) * 0.00002,
	}
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		log.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
