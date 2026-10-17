package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

const retryDelay = 2 * time.Second

type StepProcessor interface {
	ProcessStep(ctx context.Context, step domain.AgentStep) error
}

type slogLogger struct {
	log *slog.Logger
}

func (l slogLogger) Printf(msg string, args ...any) {
	l.log.Debug(fmt.Sprintf(msg, args...))
}

type Consumer struct {
	reader    *kafka.Reader
	processor StepProcessor
	log       *slog.Logger
}

type ConsumerConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	StartOffset       string
	MinBytes          int
	MaxBytes          int
	MaxWait           time.Duration
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
}

func NewConsumer(cfg ConsumerConfig, processor StepProcessor, log *slog.Logger) *Consumer {
	startOffset := kafka.LastOffset
	if cfg.StartOffset == "first" {
		startOffset = kafka.FirstOffset
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		GroupID:           cfg.GroupID,
		Topic:             cfg.Topic,
		StartOffset:       startOffset,
		CommitInterval:    0, // manual synchronous commit: At-Least-Once
		MinBytes:          cfg.MinBytes,
		MaxBytes:          cfg.MaxBytes,
		MaxWait:           cfg.MaxWait,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
		Logger:            slogLogger{log: log},
		ErrorLogger:       slogLogger{log: log},
	})
	return &Consumer{reader: reader, processor: processor, log: log}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Run reads messages until ctx is cancelled. The offset is committed only
// after the step has been successfully persisted to both databases.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}

		step, err := domain.ParseStep(msg.Value)
		if err != nil {
			c.log.Error("poison message skipped", "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return err
			}
			continue
		}

		if err := c.processWithRetry(ctx, step); err != nil {
			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
		c.log.Info("step processed",
			"run_id", step.RunID,
			"step_number", step.StepNumber,
			"status", step.Status,
			"tool_called", step.ToolCalled,
		)
	}
}

func (c *Consumer) processWithRetry(ctx context.Context, step domain.AgentStep) error {
	for {
		err := c.processor.ProcessStep(ctx, step)
		if err == nil {
			return nil
		}
		c.log.Error("process step failed, retrying",
			"run_id", step.RunID,
			"step_number", step.StepNumber,
			"error", err,
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
		}
	}
}
