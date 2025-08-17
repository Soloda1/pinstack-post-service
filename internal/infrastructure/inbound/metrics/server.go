package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	ports "pinstack-post-service/internal/domain/ports/output"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	server  *http.Server
	address string
	port    int
	log     ports.Logger
}

func NewMetricsServer(address string, port int, log ports.Logger) *Server {
	return &Server{
		address: address,
		port:    port,
		log:     log,
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.address, s.port)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.log.Info("Starting Prometheus metrics server", slog.String("address", addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.log.Info("Shutting down metrics server")
	return s.server.Shutdown(ctx)
}
