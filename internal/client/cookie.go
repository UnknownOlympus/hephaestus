package client

import (
	"log/slog"
	"net/http"
	"net/url"
	"sync"
)

// CookieJar implements http.CookieJar for storing cookies in memory.
type CookieJar struct {
	log *slog.Logger
	mu  sync.Mutex
	jar map[string][]*http.Cookie
}

// NewCookieJar initializes an in-memory cookie jar.
func NewCookieJar(log *slog.Logger) *CookieJar {
	return &CookieJar{
		jar: make(map[string][]*http.Cookie),
		log: log,
		mu:  sync.Mutex{},
	}
}

// SetCookies stores cookies for a given URL.
func (c *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jar[u.Host] = cookies
	c.log.Debug("Set cookies", "host", u.Host)
}

// Cookies retrieves cookies for a given URL.
func (c *CookieJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.jar[u.Host]
}
