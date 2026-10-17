package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/domain"
	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/usecase"
)

const (
	nodeType   = "AgentStepNode"
	nodeWidth  = 200
	nodeHeight = 170
)

type Handler struct {
	queries *usecase.TrajectoryQuery
	log     *slog.Logger
}

func NewHandler(queries *usecase.TrajectoryQuery, log *slog.Logger) *Handler {
	return &Handler{queries: queries, log: log}
}

func (h *Handler) Routes(basePath string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/runs/{run_id}/trajectory", h.getTrajectory)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return withCORS(mux)
}

func (h *Handler) getTrajectory(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	traj, err := h.queries.GetTrajectory(r.Context(), runID)
	if err != nil {
		if errors.Is(err, usecase.ErrRunNotFound) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		h.log.Error("get trajectory", "run_id", runID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toFlowResponse(traj))
}

func toFlowResponse(traj domain.Trajectory) flowResponse {
	resp := flowResponse{
		Nodes: make([]flowNode, 0, len(traj.Nodes)),
		Edges: make([]flowEdge, 0, len(traj.Edges)),
	}
	for _, n := range traj.Nodes {
		resp.Nodes = append(resp.Nodes, flowNode{
			ID:   strconv.Itoa(n.Number),
			Type: nodeType,
			Position: position{
				X: float64(nodeWidth),
				Y: float64((n.Number - 1) * nodeHeight),
			},
			Data: flowNodeData{
				StepNumber: n.Number,
				Thought:    n.Thought,
				ToolCalled: n.ToolCalled,
				Status:     string(n.Status),
				LatencyMs:  n.LatencyMs,
				TokensUsed: n.TokensUsed,
			},
		})
	}
	for _, e := range traj.Edges {
		resp.Edges = append(resp.Edges, flowEdge{
			ID:       fmt.Sprintf("e-%d-%d", e.From, e.To),
			Source:   strconv.Itoa(e.From),
			Target:   strconv.Itoa(e.To),
			Animated: e.Animated,
		})
	}
	return resp
}

type flowResponse struct {
	Nodes []flowNode `json:"nodes"`
	Edges []flowEdge `json:"edges"`
}

type position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type flowNode struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Position position     `json:"position"`
	Data     flowNodeData `json:"data"`
}

type flowNodeData struct {
	StepNumber int    `json:"stepNumber"`
	Thought    string `json:"thought"`
	ToolCalled string `json:"toolCalled"`
	Status     string `json:"status"`
	LatencyMs  int    `json:"latencyMs"`
	TokensUsed int    `json:"tokensUsed"`
}

type flowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Animated bool   `json:"animated"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
