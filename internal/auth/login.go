package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Houeta/us-api-provider/internal/models"
)

var ErrLogin = errors.New("login failed")

// Login performs a login request to the specified loginURL using the provided username and password.
// It returns an error if the request fails or the response status code is not 200 OK.
func Login(ctx context.Context, client *http.Client, loginURL, baseURL, username, password string) error {
	// Data for login
	data := url.Values{}
	data.Set("action", "login")
	data.Set("username", username)
	data.Set("password", password)

	// Create a POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create new request %s: %w", loginURL, err)
	}

	// Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", models.UserAgent)
	req.Header.Set("Referer", baseURL)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w, status code: %d", ErrLogin, resp.StatusCode)
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	return nil
}

func RetryLogin(
	ctx context.Context,
	log *slog.Logger,
	httpClient *http.Client,
	loginURL, baseURL, username, password string,
) error {
	var err error

	const retryTimeout = 5 * time.Second
	const retries = 3

	for index := range retries {
		err = Login(ctx, httpClient, loginURL, baseURL, username, password)
		if err == nil {
			log.InfoContext(ctx, "Successfuly logged in")
			return nil
		}

		log.WarnContext(ctx, "Failed to login, retrying...", "attempt", index+1, "of", retries, "error", err.Error())
		time.Sleep(retryTimeout)
	}

	finalError := errors.New("failed to login after multiple retries")
	log.ErrorContext(ctx, finalError.Error(), "last_error", err)
	return finalError
}
