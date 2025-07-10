package server_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Houeta/us-api-provider/internal/server"
	"github.com/stretchr/testify/require"
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("all systems ok", func(t *testing.T) {
		mockParserServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mockParserServer.Close()

		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(mockDB, mockParserServer.URL, logger)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		expectedBody := `{"database":"ok","parser_host":"ok"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("database unavailable", func(t *testing.T) {
		mockParserServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mockParserServer.Close()

		mockDB := &MockDBPinger{ShouldFail: true}
		healthChecker := server.NewHealthChecker(mockDB, mockParserServer.URL, logger)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"unavailable","parser_host":"ok"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("parser host unavailable", func(t *testing.T) {
		mockParserServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockParserServer.Close()

		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(mockDB, mockParserServer.URL, logger)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"ok","parser_host":"degraded"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})

	t.Run("parser host runreachable", func(t *testing.T) {
		mockDB := &MockDBPinger{ShouldFail: false}
		healthChecker := server.NewHealthChecker(mockDB, "invalid_url", logger)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		healthChecker.ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		expectedBody := `{"database":"ok","parser_host":"unreachable"}`
		require.JSONEq(t, expectedBody, rr.Body.String())
	})
}
