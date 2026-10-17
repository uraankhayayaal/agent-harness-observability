package domain

type TrajectoryNode struct {
	Number     int
	Thought    string
	ToolCalled string
	Status     StepStatus
	LatencyMs  int
	TokensUsed int
}

type TrajectoryEdge struct {
	From     int
	To       int
	Animated bool
}

type Trajectory struct {
	Nodes []TrajectoryNode
	Edges []TrajectoryEdge
}
