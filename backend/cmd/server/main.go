package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpapi "github.com/uraankhayayaal/agent-harness-observability/backend/internal/adapter/http"
	kafkaadapter "github.com/uraankhayayaal/agent-harness-observability/backend/internal/adapter/kafka"
	neo4jadapter "github.com/uraankhayayaal/agent-harness-observability/backend/internal/adapter/neo4j"
	postgresadapter "github.com/uraankhayayaal/agent-harness-observability/backend/internal/adapter/postgres"
	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/config"
	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/usecase"
)

func main() {
	log := newLogger()
	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func run(log *slog.Logger) error {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pgRepo, err := postgresadapter.New(ctx, cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer pgRepo.Close()
	if err := pgRepo.EnsureSchema(ctx); err != nil {
		return err
	}

	graphRepo, err := neo4jadapter.New(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		return err
	}
	defer func() {
		if err := graphRepo.Close(context.Background()); err != nil {
			log.Error("close neo4j", "error", err)
		}
	}()

	processor := usecase.NewStepProcessor(pgRepo, graphRepo)
	queries := usecase.NewTrajectoryQuery(graphRepo)

	consumer := kafkaadapter.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, processor, log)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("close kafka consumer", "error", err)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	srv := httpapi.NewServer(cfg.HTTPAddr, queries, log)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errCh:
		stop()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
