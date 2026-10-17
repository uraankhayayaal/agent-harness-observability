-- test_runs: metadata of agent test sessions
CREATE TABLE IF NOT EXISTS test_runs (
    id UUID PRIMARY KEY,
    harness_name VARCHAR(255) NOT NULL,
    agent_version VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'running'
);

-- run_metrics: aggregated per-run counters
CREATE TABLE IF NOT EXISTS run_metrics (
    run_id UUID PRIMARY KEY REFERENCES test_runs(id) ON DELETE CASCADE,
    total_steps INT NOT NULL DEFAULT 0,
    total_tokens_used INT NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC(10, 5) NOT NULL DEFAULT 0.0,
    execution_time_ms INT NOT NULL DEFAULT 0
);
