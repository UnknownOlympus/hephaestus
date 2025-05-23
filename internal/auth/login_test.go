package auth_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Houeta/us-api-provider/internal/auth"
	"github.com/Houeta/us-api-provider/internal/models"
)

type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func (er *errorReader) Close() error {
	return nil
}

// mockRoundTripper helps to imitate errors on transoprt and custom request level
type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc == nil {
		return nil, errors.New("RoundTripFunc not set")
	}
	return m.RoundTripFunc(req)
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name                  string
		serverHandlerFactory  func(baseURL string) http.HandlerFunc
		loginURLOverride      string
		clientTransport       http.RoundTripper
		ctx                   context.Context
		username              string
		password              string
		baseURL               string
		wantErr               bool
		expectedSpecificError error
		wantErrMsgContains    string
	}{
		{
			name: "success login",
			serverHandlerFactory: func(expectedBaseURL string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// check method
					if r.Method != http.MethodPost {
						t.Errorf("Expected POST method, but received %s", r.Method)
						http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
						return
					}

					// check headers
					if got, want := r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
						t.Errorf("Expected Content-Type '%s', but received '%s'", want, got)
					}
					if got, want := r.Header.Get("User-Agent"), models.UserAgent; got != want {
						t.Errorf("Expected User-Agent '%s', but received '%s'", want, got)
					}
					if got, want := r.Header.Get("Referer"), expectedBaseURL; got != want {
						t.Errorf("Expected Referer '%s', but received '%s'", want, got)
					}

					// check form data
					if err := r.ParseForm(); err != nil {
						http.Error(w, "failed to parse the form", http.StatusBadRequest)
						return
					}
					if got, want := r.FormValue("action"), "login"; got != want {
						t.Errorf("Expected action 'login', but received '%s'", got)
					}
					if got, want := r.FormValue("username"), "testuser"; got != want {
						t.Errorf("Expected username 'testuser', but received '%s'", got)
					}
					if got, want := r.FormValue("password"), "testpass"; got != want {
						t.Errorf("Expected password 'testpass', but received '%s'", got)
					}
					w.WriteHeader(http.StatusOK)
					fmt.Println(w, "Login successful")
				}
			},
			ctx:      context.Background(),
			username: "testuser",
			password: "testpass",
			baseURL:  "http://example.com",
			wantErr:  false,
		},
		{
			name:               "error creating new request - invalid URL",
			loginURLOverride:   "http://invalid url bla bla bla",
			ctx:                context.Background(),
			username:           "testuser",
			password:           "testpass",
			baseURL:            "http://example.com",
			wantErr:            true,
			wantErrMsgContains: "failed to create new request",
		},
		{
			name: "request execution error - client.Do error",
			clientTransport: &mockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("simulated network error")
				},
			},
			ctx:                context.Background(),
			username:           "testuser",
			password:           "testpass",
			baseURL:            "http://example.com",
			wantErr:            true,
			wantErrMsgContains: "failed to request",
		},
		{
			name: "failed to login; status code != 200",
			serverHandlerFactory: func(_ string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Println(w, "not authorized")
				}
			},
			ctx:                   context.Background(),
			username:              "testuser",
			password:              "testpass",
			baseURL:               "http://example.com",
			wantErr:               true,
			expectedSpecificError: auth.ErrLogin,
			wantErrMsgContains:    fmt.Sprintf("status code: %d", http.StatusUnauthorized),
		},
		{
			name: "error reading response body",
			clientTransport: &mockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       &errorReader{},
						Header:     make(http.Header),
					}, nil
				},
			},
			ctx:                context.Background(),
			username:           "testuser",
			password:           "testpass",
			baseURL:            "http://example.com",
			wantErr:            true,
			wantErrMsgContains: "failed to read response body: simulated read error",
		},
		{
			name: "context was canceled before the request was called",
			serverHandlerFactory: func(baseURL string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					t.Error("The server handler should not be called if the context is canceled before the request")
				}
			},
			ctx: func() context.Context {
				c, cancel := context.WithCancel(context.Background())
				cancel() // cancel immediately
				return c
			}(),
			username:              "testuser",
			password:              "testpass",
			baseURL:               "http://example.com",
			wantErr:               true,
			expectedSpecificError: context.Canceled,
			wantErrMsgContains:    "failed to request",
		},
		{
			name: "context timeout while request is executing",
			clientTransport: &mockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					select {
					case <-time.After(100 * time.Millisecond):
						return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
					case <-req.Context().Done():
						return nil, req.Context().Err()
					}
				},
			},
			ctx: func() context.Context {
				// timeout shorter than simulated transport delay
				c, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				_ = cancel
				return c
			}(),
			username:              "testuser",
			password:              "testpass",
			baseURL:               "http://example.com",
			wantErr:               true,
			expectedSpecificError: context.DeadlineExceeded,
			wantErrMsgContains:    "failed to request",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testServer *httptest.Server
			actualLoginURL := "http://dummy.example.com/login"

			// setup test server, if needed
			if tt.serverHandlerFactory != nil {
				testServer = httptest.NewServer(tt.serverHandlerFactory(tt.baseURL))
				defer testServer.Close()
				actualLoginURL = testServer.URL // use URL of test server
			}
			// using the override url, if provided
			if tt.loginURLOverride != "" {
				actualLoginURL = tt.loginURLOverride
			}

			// configure http client
			httpClient := &http.Client{} // client by default
			if testServer != nil && tt.clientTransport != nil {
				// use the test server client if there is a server and there is no custom transport
				httpClient = testServer.Client()
			}
			if tt.clientTransport != nil {
				// use the custom transport, if provided (has priority)
				httpClient.Transport = tt.clientTransport
			}

			// Call Login function
			err := auth.Login(tt.ctx, httpClient, actualLoginURL, tt.baseURL, tt.username, tt.password)

			// Check results
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Login() expected an error but received nil")
				}
				// Check on spesific error type (if provided)
				if tt.expectedSpecificError != nil {
					if !errors.Is(err, tt.expectedSpecificError) {
						t.Errorf("Login() error type = %T (%v), expected wrap %T (%v)",
							err, err, tt.expectedSpecificError, tt.expectedSpecificError)
					}
				}
				// Check for error message content (if specified)
				if tt.wantErrMsgContains != "" && !strings.Contains(err.Error(), tt.wantErrMsgContains) {
					t.Errorf("Login() error = %q, expected contains %q", err.Error(), tt.wantErrMsgContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Login() unexpected error: %v", err)
				}
			}
		})
	}
}
