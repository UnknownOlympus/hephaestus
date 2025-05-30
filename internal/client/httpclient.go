package client

import (
	"log/slog"
	"net/http"
)

// CreateHTTPClient initializes an HTTP client with a custom cookie jar.
func CreateHTTPClient(log *slog.Logger) *http.Client {
	jar := NewCookieJar(log)

	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			log.Debug("Redirected to URL", "URL", req.URL)

			return nil
		},
	}
}
