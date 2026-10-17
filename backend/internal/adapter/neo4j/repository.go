package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
)

// saveStepCypher is idempotent: reprocessing the same (run_id, step_number)
// updates the existing Step node instead of creating a duplicate.
const saveStepCypher = `
MERGE (r:Run {id: $run_id})
ON CREATE SET r.agent = $agent_version
MERGE (s:Step {run_id: $run_id, number: $step_number})
SET s.thought = $thought,
    s.tool_called = $tool_called,
    s.status = $status,
    s.latency_ms = $latency_ms,
    s.tokens_used = $tokens_used
WITH r, s
FOREACH (_ IN CASE WHEN $step_number = 1 THEN [1] ELSE [] END |
    MERGE (r)-[:HAS_FIRST_STEP]->(s)
)
WITH r, s
OPTIONAL MATCH (prev:Step {run_id: $run_id, number: $step_number - 1})
FOREACH (_ IN CASE WHEN prev IS NOT NULL THEN [1] ELSE [] END |
    MERGE (prev)-[:NEXT_STEP]->(s)
)
RETURN s.number AS number
`

const loadNodesCypher = `
MATCH (r:Run {id: $run_id})-[:HAS_FIRST_STEP]->(f:Step)
MATCH path = (f)-[:NEXT_STEP*0..50]->(m)
UNWIND nodes(path) AS n
WITH DISTINCT n
ORDER BY n.number
RETURN n.number AS number,
       n.thought AS thought,
       n.tool_called AS tool_called,
       n.status AS status,
       n.latency_ms AS latency_ms,
       n.tokens_used AS tokens_used
`

const loadEdgesCypher = `
MATCH (r:Run {id: $run_id})-[:HAS_FIRST_STEP]->(f:Step)
MATCH path = (f)-[:NEXT_STEP*0..50]->(m)
UNWIND relationships(path) AS rel
WITH DISTINCT rel
RETURN startNode(rel).number AS from_number,
       endNode(rel).number AS to_number,
       startNode(rel).status AS from_status,
       startNode(rel).tool_called AS from_tool
`

type Repository struct {
	driver neo4j.DriverWithContext
}

func New(ctx context.Context, uri, user, password string) (*Repository, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}
	return &Repository{driver: driver}, nil
}

func (r *Repository) Close(ctx context.Context) error {
	return r.driver.Close(ctx)
}

func (r *Repository) SaveStep(ctx context.Context, step domain.AgentStep) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, saveStepCypher, map[string]any{
			"run_id":        step.RunID,
			"agent_version": step.AgentVersion,
			"step_number":   step.StepNumber,
			"thought":       step.Thought,
			"tool_called":   step.ToolCalled,
			"status":        string(step.Status),
			"latency_ms":    step.LatencyMs,
			"tokens_used":   step.TokensUsed,
		})
		if err != nil {
			return nil, err
		}
		if result.Next(ctx) {
			return nil, nil
		}
		return nil, result.Err()
	})
	if err != nil {
		return fmt.Errorf("execute write: %w", err)
	}
	return nil
}

func (r *Repository) LoadTrajectory(ctx context.Context, runID string) (domain.Trajectory, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	nodes, err := loadNodes(ctx, session, runID)
	if err != nil {
		return domain.Trajectory{}, err
	}
	edges, err := loadEdges(ctx, session, runID)
	if err != nil {
		return domain.Trajectory{}, err
	}
	return domain.Trajectory{Nodes: nodes, Edges: edges}, nil
}

func loadNodes(ctx context.Context, session neo4j.SessionWithContext, runID string) ([]domain.TrajectoryNode, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, loadNodesCypher, map[string]any{"run_id": runID})
		if err != nil {
			return nil, err
		}
		var nodes []domain.TrajectoryNode
		for res.Next(ctx) {
			rec := res.Record()
			node := domain.TrajectoryNode{
				Number:     asInt(rec.Values[0]),
				Thought:    asString(rec.Values[1]),
				ToolCalled: asString(rec.Values[2]),
				Status:     domain.StepStatus(asString(rec.Values[3])),
				LatencyMs:  asInt(rec.Values[4]),
				TokensUsed: asInt(rec.Values[5]),
			}
			nodes = append(nodes, node)
		}
		return nodes, res.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("load nodes: %w", err)
	}
	nodes, _ := result.([]domain.TrajectoryNode)
	return nodes, nil
}

func loadEdges(ctx context.Context, session neo4j.SessionWithContext, runID string) ([]domain.TrajectoryEdge, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, loadEdgesCypher, map[string]any{"run_id": runID})
		if err != nil {
			return nil, err
		}
		var edges []domain.TrajectoryEdge
		for res.Next(ctx) {
			rec := res.Record()
			edge := domain.TrajectoryEdge{
				From:     asInt(rec.Values[0]),
				To:       asInt(rec.Values[1]),
				Animated: asString(rec.Values[2]) == string(domain.StepStatusSuccess) && asString(rec.Values[3]) != "",
			}
			edges = append(edges, edge)
		}
		return edges, res.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("load edges: %w", err)
	}
	edges, _ := result.([]domain.TrajectoryEdge)
	return edges, nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
