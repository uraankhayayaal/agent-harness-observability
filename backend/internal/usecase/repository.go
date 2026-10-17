package usecase

import (
	"context"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

type RunStore interface {
	UpsertRunMetadata(ctx context.Context, step domain.AgentStep) error
	IncrementMetrics(ctx context.Context, step domain.AgentStep) error
}

type GraphStore interface {
	SaveStep(ctx context.Context, step domain.AgentStep) error
	LoadTrajectory(ctx context.Context, runID string) (domain.Trajectory, error)
}
