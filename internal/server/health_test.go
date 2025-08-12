package server_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/server"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

type MockDBPinger struct {
	ShouldFail bool
}

func (m *MockDBPinger) Ping(_ context.Context) error {
	if m.ShouldFail {
		return errors.New("mock db error")
	}
	return nil
}

func TestHealthChecker(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("all systems ok", func(t *testing.T) {
		t.Parallel()

		lis := bufconn.Listen(1024 * 1024)
		s := grpc.NewServer()
		defer s.GracefulStop()
		healthSrv := health.NewServer()
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		grpc_health_v1.RegisterHealthServer(s, healthSrv)
		go func() {
			if err := s.Serve(lis); err != nil {
				slog.Error("Test server failed", "error", err)
			}
		}()

		conn, err := grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(logger, mockDB, conn)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		expectedBody := `{"database":"ok", "hermes_service":"ok"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("database unavailable", func(t *testing.T) {
		t.Parallel()

		lis := bufconn.Listen(1024 * 1024)
		s := grpc.NewServer()
		defer s.GracefulStop()
		healthSrv := health.NewServer()
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		grpc_health_v1.RegisterHealthServer(s, healthSrv)
		go func() { _ = s.Serve(lis) }()

		conn, err := grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		mockDB := &MockDBPinger{ShouldFail: true}
		healthChecker := server.NewHealthChecker(logger, mockDB, conn)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"unavailable", "hermes_service":"ok"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("hermes service degraded", func(t *testing.T) {
		t.Parallel()

		lis := bufconn.Listen(1024 * 1024)
		s := grpc.NewServer()
		defer s.GracefulStop()
		healthSrv := health.NewServer()
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		grpc_health_v1.RegisterHealthServer(s, healthSrv)
		go func() { _ = s.Serve(lis) }()

		conn, err := grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(logger, mockDB, conn)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"ok", "hermes_service":"degraded"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("hermes service unreachable", func(t *testing.T) {
		lis := bufconn.Listen(1024 * 1024)
		conn, err := grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		lis.Close()
		defer conn.Close()

		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(logger, mockDB, conn)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"ok", "hermes_service":"unreachable"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})
}
