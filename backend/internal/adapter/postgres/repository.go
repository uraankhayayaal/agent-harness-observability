package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
	"github.com/uraankhayayaal/agent-harness-observability/backend/migrations"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) EnsureSchema(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, migrations.InitSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) UpsertRunMetadata(ctx context.Context, step domain.AgentStep) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO test_runs (id, harness_name, agent_version)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, step.RunID, step.HarnessName, step.AgentVersion)
	if err != nil {
		return fmt.Errorf("insert test_runs: %w", err)
	}
	return nil
}

func (r *Repository) IncrementMetrics(ctx context.Context, step domain.AgentStep) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO run_metrics (run_id, total_steps, total_tokens_used, total_cost_usd, execution_time_ms)
		VALUES ($1, 1, $2, $3, $4)
		ON CONFLICT (run_id) DO UPDATE SET
			total_steps = run_metrics.total_steps + 1,
			total_tokens_used = run_metrics.total_tokens_used + EXCLUDED.total_tokens_used,
			total_cost_usd = run_metrics.total_cost_usd + EXCLUDED.total_cost_usd,
			execution_time_ms = run_metrics.execution_time_ms + EXCLUDED.execution_time_ms
	`, step.RunID, step.TokensUsed, step.CostUSD, step.LatencyMs)
	if err != nil {
		return fmt.Errorf("upsert run_metrics: %w", err)
	}
	return nil
}
