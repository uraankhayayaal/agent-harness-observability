package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/uraankhayayaal/agent-harness-observability/backend/internal/usecase"
)

type Server struct {
	http *http.Server
	log  *slog.Logger
}

func NewServer(addr, basePath string, queries *usecase.TrajectoryQuery, log *slog.Logger) *Server {
	h := NewHandler(queries, log)
	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           h.Routes(basePath),
			ReadHeaderTimeout: 5 * time.Second,
		},
		log: log,
	}
}

func (s *Server) ListenAndServe() error {
	s.log.Info("http server listening", "addr", s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
