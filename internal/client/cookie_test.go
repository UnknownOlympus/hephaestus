package client_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/Houeta/us-api-provider/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCookies(t *testing.T) {
	t.Parallel()

	reqURL, err := url.Parse("http://example.com")
	require.NoError(t, err)

	expectedCookies := []*http.Cookie{
		{
			Name:   "test",
			Value:  "testValue",
			Quoted: false,
		},
	}
	logger := slog.New(slog.Default().Handler())
	cookie := client.NewCookieJar(logger)
	cookie.SetCookies(reqURL, expectedCookies)
}

func TestCookies(t *testing.T) {
	t.Parallel()

	reqURL, err := url.Parse("http://example.com")
	require.NoError(t, err)

	expectedCookies := []*http.Cookie{
		{
			Name:   "test",
			Value:  "testValue",
			Quoted: false,
		},
	}
	logger := slog.New(slog.Default().Handler())
	cookie := client.NewCookieJar(logger)
	cookie.SetCookies(reqURL, expectedCookies)

	actualCookie := cookie.Cookies(reqURL)
	assert.Equal(t, expectedCookies, actualCookie)
}
