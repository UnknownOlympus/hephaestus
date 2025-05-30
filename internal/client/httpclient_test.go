package client_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Houeta/us-api-provider/internal/client"
)

func TestCreateHTTPClient(t *testing.T) {
	var logBuf bytes.Buffer // buffer for log capturing
	// Create slog.Logger, which writes in logBuf
	testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Level debug needed, for CheckRedirect message capturing
	}))

	t.Run("client properties", func(t *testing.T) {
		client := client.CreateHTTPClient(testLogger) // Call function which testing

		if client.Jar == nil {
			t.Error("client.Jar must be initiated and must not be nil")
		}

		if client.CheckRedirect == nil {
			t.Error("client.CheckRedirect must be set and must not be nil")
		}
	})

	t.Run("CheckRedirect behavior - redirection and logging", func(t *testing.T) {
		logBuf.Reset() // Clearing the log buffer before this particular test

		finalPath := "/final-destination"
		redirectPath := "/redirect-here"

		// Setup test server, which redirects
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case redirectPath:
				http.Redirect(w, r, finalPath, http.StatusFound)
			case finalPath:
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Endpoint successfully reached"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		// Create a client using the function we are testing
		client := client.CreateHTTPClient(testLogger)

		// Making a request tahat initiates a redirect
		resp, err := client.Get(server.URL + redirectPath)
		if err != nil {
			t.Fatalf("client.Get failed: %v", err)
		}
		defer resp.Body.Close()

		// 1. Check if redirect was completed
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK (200) after redirect, but received %d", resp.StatusCode)
		}
		if resp.Request.URL.Path != finalPath {
			t.Errorf("Expected request final path %s, but received %s", finalPath, resp.Request.URL.Path)
		}

		// 2. Check logging with CheckRedirect
		loggedOutput := logBuf.String()

		expectedLoggedMsgPart := "Redirected to URL"
		// Expected logger URL value - its an absoloute URL path on the test server.
		expectedLoggedURLValue := server.URL + finalPath

		if !strings.Contains(loggedOutput, expectedLoggedMsgPart) {
			t.Errorf(
				"The log output does not contain the expected part of message %q. Log:\n%s",
				expectedLoggedMsgPart,
				loggedOutput,
			)
		}

		// slog text log format: level=DEBUG msg="Redirected to URL" URL="http://127.0.0.1:xxxx/final-destination"
		expectedLoggedURLAttribute := "URL=" + expectedLoggedURLValue
		if !strings.Contains(loggedOutput, expectedLoggedURLAttribute) {
			t.Errorf(
				"The log output does not contain the expected URL attribute %s. Log:\n%s",
				expectedLoggedURLAttribute,
				loggedOutput,
			)
		}
	})
}
