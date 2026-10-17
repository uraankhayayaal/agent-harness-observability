package usecase

import (
	"context"
	"fmt"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

type StepProcessor struct {
	runs  RunStore
	graph GraphStore
}

func NewStepProcessor(runs RunStore, graph GraphStore) *StepProcessor {
	return &StepProcessor{runs: runs, graph: graph}
}

func (p *StepProcessor) ProcessStep(ctx context.Context, step domain.AgentStep) error {
	if err := step.Validate(); err != nil {
		return fmt.Errorf("invalid step: %w", err)
	}
	if err := p.runs.UpsertRunMetadata(ctx, step); err != nil {
		return fmt.Errorf("upsert run metadata: %w", err)
	}
	if err := p.runs.IncrementMetrics(ctx, step); err != nil {
		return fmt.Errorf("increment run metrics: %w", err)
	}
	if err := p.graph.SaveStep(ctx, step); err != nil {
		return fmt.Errorf("save step to graph: %w", err)
	}
	return nil
}
