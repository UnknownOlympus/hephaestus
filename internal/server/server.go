package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

// StartMonitoringServer starts an HTTP server that provides health check and metrics endpoints.
// It listens on the specified port and logs the server's status and any errors encountered.
//
// Parameters:
// - ctx: A context.Context for managing cancellation and timeouts.
// - log: A logger for logging server events and errors.
// - reg: A registry with Prometheus collectors.
// - dtb: A pgxpool connector for database methods (ping)
// - port: The port number on which the server will listen.
func StartMonitoringServer(
	ctx context.Context,
	log *slog.Logger,
	reg *prometheus.Registry,
	dtb *pgxpool.Pool,
	port int,
	hermesConn *grpc.ClientConn,
) {
	mux := http.NewServeMux()
	healthChecker := NewHealthChecker(log, dtb, hermesConn)

	mux.Handle("/healthz", healthChecker)
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	log.InfoContext(ctx, "Starting monitoring server", "port", port)

	readTimeout := 5
	writeTimeout := 10
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}

	var err error
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(readTimeout)*time.Second)
		defer cancel()
		log.InfoContext(ctx, "Monitoring server shutting down.")
		if err = server.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Monitoring server failed to shutdown", "error", err)
			return
		}
	case err = <-serverErr:
		log.ErrorContext(ctx, "Monitoring server failed", "error", err)
	}
}
