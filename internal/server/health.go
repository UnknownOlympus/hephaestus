package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthChecker struct {
	db           DBPinger
	log          *slog.Logger
	hermesHealth grpc_health_v1.HealthClient
}

func NewHealthChecker(log *slog.Logger, db DBPinger, hermesConn *grpc.ClientConn) *HealthChecker {
	return &HealthChecker{
		db:           db,
		log:          log,
		hermesHealth: grpc_health_v1.NewHealthClient(hermesConn),
	}
}

func (h *HealthChecker) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	h.log.DebugContext(req.Context(), "Performing health checks...")

	var err error
	status := make(map[string]string)
	overallStatus := http.StatusOK

	if err = h.db.Ping(req.Context()); err != nil {
		status["database"] = "unavailable"
		overallStatus = http.StatusServiceUnavailable
		h.log.WarnContext(req.Context(), "Health check failed: DB ping", "error", err)
	} else {
		status["database"] = "ok"
	}

	healthReq := &grpc_health_v1.HealthCheckRequest{Service: ""}
	resp, err := h.hermesHealth.Check(req.Context(), healthReq)
	switch {
	case err != nil:
		status["hermes_service"] = "unreachable"
		overallStatus = http.StatusServiceUnavailable
		h.log.WarnContext(req.Context(), "Health check failed: Hermes service unreachable", "error", err)
	case resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING:
		status["hermes_service"] = "degraded"
		overallStatus = http.StatusServiceUnavailable
		h.log.WarnContext(
			req.Context(),
			"Health check failed: Hermes service is not serving",
			"status",
			resp.GetStatus().String(),
		)
	default:
		status["hermes_service"] = "ok"
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(overallStatus)
	if err = json.NewEncoder(writer).Encode(status); err != nil {
		h.log.ErrorContext(req.Context(), "Failed to write health check response", "error", err)
	}

	h.log.DebugContext(req.Context(), "Health checks completed", "status", overallStatus)
}
