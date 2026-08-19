package airtel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	cachedToken string
	tokenExpiry time.Time
	tokenMutex  sync.Mutex
)

type tokenResponse struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetValidAccessToken fetches a fresh OAuth2 token or returns a valid cached one
// to avoid Airtel rate limits. Unlike the standalone this never logs the token
// response body or client_id (gateway invariant: secrets never in logs).
func GetValidAccessToken() (string, error) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// tokenExpiry already carries the refresh margin (see refreshMargin).
	if cachedToken != "" && time.Now().Before(tokenExpiry) {
		return cachedToken, nil
	}

	clientID, err := requireEnv("AIRTEL_CLIENT_ID")
	if err != nil {
		return "", err
	}
	clientSecret, err := requireEnv("AIRTEL_CLIENT_SECRET")
	if err != nil {
		return "", err
	}
	base, err := BaseURL()
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"grant_type":    "client_credentials",
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	authURL := base + "/auth/oauth2/token"
	req, err := http.NewRequest(http.MethodPost, authURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		// Do not echo the raw body — it carries the access token on success and
		// may carry credential hints on failure.
		return "", fmt.Errorf("airtel: token request failed with status %d", resp.StatusCode)
	}

	var tokenRes tokenResponse
	if err := json.Unmarshal(respBody, &tokenRes); err != nil {
		return "", err
	}
	if tokenRes.AccessToken == "" {
		return "", fmt.Errorf("airtel: received empty access token")
	}

	cachedToken = tokenRes.AccessToken
	expiresInSec := tokenRes.ExpiresIn
	if expiresInSec <= 0 {
		expiresInSec = 3600
	}
	lifetime := time.Duration(expiresInSec) * time.Second
	tokenExpiry = time.Now().Add(lifetime - refreshMargin(lifetime))

	return cachedToken, nil
}

// refreshMargin returns how long before real expiry the cached token is
// discarded. Airtel production tokens can be as short as 180s, so scale the
// margin instead of using a flat value that would waste most of the lifetime.
func refreshMargin(lifetime time.Duration) time.Duration {
	margin := lifetime / 10
	if margin < 5*time.Second {
		margin = 5 * time.Second
	}
	if margin > time.Minute {
		margin = time.Minute
	}
	if half := lifetime / 2; margin > half {
		margin = half
	}
	return margin
}
