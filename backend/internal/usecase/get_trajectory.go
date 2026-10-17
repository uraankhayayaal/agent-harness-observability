package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

var ErrRunNotFound = errors.New("run not found")

type TrajectoryQuery struct {
	graph GraphStore
}

func NewTrajectoryQuery(graph GraphStore) *TrajectoryQuery {
	return &TrajectoryQuery{graph: graph}
}

func (q *TrajectoryQuery) GetTrajectory(ctx context.Context, runID string) (domain.Trajectory, error) {
	if runID == "" {
		return domain.Trajectory{}, errors.New("run_id is required")
	}
	traj, err := q.graph.LoadTrajectory(ctx, runID)
	if err != nil {
		return domain.Trajectory{}, fmt.Errorf("load trajectory: %w", err)
	}
	if len(traj.Nodes) == 0 {
		return domain.Trajectory{}, ErrRunNotFound
	}
	return traj, nil
}
