package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthChecker struct {
	db         DBPinger
	parseHost  string
	httpClient *http.Client
	log        *slog.Logger
}

func NewHealthChecker(db DBPinger, parseHost string, log *slog.Logger) *HealthChecker {
	clientTO := 5
	return &HealthChecker{
		db:         db,
		parseHost:  parseHost,
		httpClient: &http.Client{Timeout: time.Duration(clientTO) * time.Second},
		log:        log,
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

	resp, err := h.httpClient.Head(h.parseHost) //nolint:noctx // ctx is overhead for this healthcheck
	switch {
	case err != nil:
		status["parser_host"] = "unreachable"
		overallStatus = http.StatusServiceUnavailable
		h.log.WarnContext(
			req.Context(),
			"Health check failed: parser host unreachable",
			"host",
			h.parseHost,
			"error",
			err,
		)
	case resp.StatusCode >= http.StatusBadRequest:
		status["parser_host"] = "degraded"
		overallStatus = http.StatusServiceUnavailable
		h.log.WarnContext(
			req.Context(),
			"Health check failed: parser host returned error status",
			"host",
			h.parseHost,
			"status_code",
			resp.StatusCode,
		)
	default:
		status["parser_host"] = "ok"
	}
	if resp != nil {
		if err = resp.Body.Close(); err != nil {
			h.log.WarnContext(req.Context(), "Failed to close response body", "error", err)
		}
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(overallStatus)
	if err = json.NewEncoder(writer).Encode(status); err != nil {
		h.log.ErrorContext(req.Context(), "Failed to write health check response", "error", err)
	}

	h.log.DebugContext(req.Context(), "Health checks completed", "status", overallStatus)
}
