package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type StepStatus string

const (
	StepStatusSuccess StepStatus = "success"
	StepStatusError   StepStatus = "error"
)

type AgentStep struct {
	RunID        string     `json:"run_id"`
	HarnessName  string     `json:"harness_name"`
	AgentVersion string     `json:"agent_version"`
	StepNumber   int        `json:"step_number"`
	Thought      string     `json:"thought"`
	ToolCalled   string     `json:"tool_called"`
	Status       StepStatus `json:"status"`
	LatencyMs    int        `json:"latency_ms"`
	TokensUsed   int        `json:"tokens_used"`
	CostUSD      float64    `json:"cost_usd"`
}

func ParseStep(data []byte) (AgentStep, error) {
	var step AgentStep
	if err := json.Unmarshal(data, &step); err != nil {
		return AgentStep{}, fmt.Errorf("decode step payload: %w", err)
	}
	if err := step.Validate(); err != nil {
		return AgentStep{}, err
	}
	return step, nil
}

func (s AgentStep) Validate() error {
	if !uuidRe.MatchString(s.RunID) {
		return fmt.Errorf("run_id must be a valid UUID, got %q", s.RunID)
	}
	if s.StepNumber < 1 {
		return errors.New("step_number must be >= 1")
	}
	if s.Status != StepStatusSuccess && s.Status != StepStatusError {
		return fmt.Errorf("invalid status: %q", s.Status)
	}
	if s.LatencyMs < 0 || s.TokensUsed < 0 || s.CostUSD < 0 {
		return errors.New("latency_ms, tokens_used and cost_usd must be non-negative")
	}
	return nil
}

func (s AgentStep) SuccessfulToolCall() bool {
	return s.Status == StepStatusSuccess && s.ToolCalled != ""
}
