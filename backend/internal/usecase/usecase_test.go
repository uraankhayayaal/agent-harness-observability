package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/usecase"
)

type fakeRunStore struct {
	upsertCalls  int
	incrementErr error
}

func (f *fakeRunStore) UpsertRunMetadata(context.Context, domain.AgentStep) error {
	f.upsertCalls++
	return nil
}

func (f *fakeRunStore) IncrementMetrics(context.Context, domain.AgentStep) error {
	return f.incrementErr
}

type fakeGraphStore struct {
	saved      []domain.AgentStep
	saveErr    error
	trajectory domain.Trajectory
	loadErr    error
}

func (f *fakeGraphStore) SaveStep(_ context.Context, step domain.AgentStep) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, step)
	return nil
}

func (f *fakeGraphStore) LoadTrajectory(context.Context, string) (domain.Trajectory, error) {
	if f.loadErr != nil {
		return domain.Trajectory{}, f.loadErr
	}
	return f.trajectory, nil
}

func validStep() domain.AgentStep {
	return domain.AgentStep{
		RunID:        "8b7d3e2a-1c4f-4e9a-9f2b-6d5c4a3b2e10",
		HarnessName:  "harness",
		AgentVersion: "v1",
		StepNumber:   1,
		Thought:      "thinking",
		Status:       domain.StepStatusSuccess,
		LatencyMs:    10,
		TokensUsed:   100,
		CostUSD:      0.002,
	}
}

func TestProcessStepPersistsToBothStores(t *testing.T) {
	runs := &fakeRunStore{}
	graph := &fakeGraphStore{}
	p := usecase.NewStepProcessor(runs, graph)

	if err := p.ProcessStep(context.Background(), validStep()); err != nil {
		t.Fatalf("ProcessStep: %v", err)
	}
	if runs.upsertCalls != 1 {
		t.Errorf("upsert calls = %d, want 1", runs.upsertCalls)
	}
	if len(graph.saved) != 1 {
		t.Fatalf("graph.saved = %d, want 1", len(graph.saved))
	}
}

func TestProcessStepRejectsInvalidStep(t *testing.T) {
	p := usecase.NewStepProcessor(&fakeRunStore{}, &fakeGraphStore{})
	step := validStep()
	step.StepNumber = 0
	if err := p.ProcessStep(context.Background(), step); err == nil {
		t.Fatal("expected error for invalid step")
	}
}

func TestProcessStepPropagatesStoreErrors(t *testing.T) {
	storeErr := errors.New("db down")

	runs := &fakeRunStore{incrementErr: storeErr}
	p := usecase.NewStepProcessor(runs, &fakeGraphStore{})
	if err := p.ProcessStep(context.Background(), validStep()); !errors.Is(err, storeErr) {
		t.Errorf("err = %v, want %v", err, storeErr)
	}

	graph := &fakeGraphStore{saveErr: storeErr}
	p = usecase.NewStepProcessor(&fakeRunStore{}, graph)
	if err := p.ProcessStep(context.Background(), validStep()); !errors.Is(err, storeErr) {
		t.Errorf("err = %v, want %v", err, storeErr)
	}
}

func TestGetTrajectory(t *testing.T) {
	traj := domain.Trajectory{
		Nodes: []domain.TrajectoryNode{{Number: 1, Status: domain.StepStatusSuccess}},
		Edges: []domain.TrajectoryEdge{{From: 1, To: 2}},
	}
	q := usecase.NewTrajectoryQuery(&fakeGraphStore{trajectory: traj})
	got, err := q.GetTrajectory(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("GetTrajectory: %v", err)
	}
	if len(got.Nodes) != 1 || len(got.Edges) != 1 {
		t.Errorf("unexpected trajectory: %+v", got)
	}
}

func TestGetTrajectoryNotFound(t *testing.T) {
	q := usecase.NewTrajectoryQuery(&fakeGraphStore{})
	_, err := q.GetTrajectory(context.Background(), "missing")
	if !errors.Is(err, usecase.ErrRunNotFound) {
		t.Errorf("err = %v, want ErrRunNotFound", err)
	}
}

func TestParseStep(t *testing.T) {
	payload := []byte(`{"run_id":"8b7d3e2a-1c4f-4e9a-9f2b-6d5c4a3b2e10","harness_name":"h","agent_version":"v","step_number":2,"thought":"t","tool_called":"bash","status":"success","latency_ms":50,"tokens_used":10,"cost_usd":0.001}`)
	step, err := domain.ParseStep(payload)
	if err != nil {
		t.Fatalf("ParseStep: %v", err)
	}
	if step.ToolCalled != "bash" || step.Status != domain.StepStatusSuccess {
		t.Errorf("unexpected step: %+v", step)
	}
	if !step.SuccessfulToolCall() {
		t.Error("SuccessfulToolCall() = false, want true")
	}

	if _, err := domain.ParseStep([]byte(`{"run_id":"8b7d3e2a-1c4f-4e9a-9f2b-6d5c4a3b2e10","step_number":0,"status":"success"}`)); err == nil {
		t.Error("expected validation error for step_number=0")
	}
	if _, err := domain.ParseStep([]byte(`{"run_id":"not-a-uuid","step_number":1,"status":"success"}`)); err == nil {
		t.Error("expected validation error for invalid run_id")
	}
	if _, err := domain.ParseStep([]byte(`not-json`)); err == nil {
		t.Error("expected decode error")
	}
}
