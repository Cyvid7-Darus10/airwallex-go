package airwallex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// expiresAtFormats covers the timestamp shapes Airwallex has been observed to
// return, including the non-RFC3339 "+0000" zone offset.
var expiresAtFormats = []string{
	"2006-01-02T15:04:05Z0700",
	"2006-01-02T15:04:05.999999999Z0700",
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999999999",
}

// parseExpiresAt converts the login response expires_at into a time.Time,
// falling back to the documented ~30 minute TTL when unparseable.
func parseExpiresAt(raw string, now time.Time) time.Time {
	if raw == "" {
		return now.Add(fallbackTokenTTL)
	}
	for _, format := range expiresAtFormats {
		if parsed, err := time.Parse(format, raw); err == nil {
			return parsed
		}
	}
	return now.Add(fallbackTokenTTL)
}

// tokenManager fetches and caches the bearer token. It is safe for
// concurrent use; a single login is performed at a time and the token is
// refreshed one minute before it expires so an in-flight request never
// carries a token that expires mid-request.
type tokenManager struct {
	clientID string
	apiKey   string
	loginURL string
	now      func() time.Time

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newTokenManager(clientID, apiKey, baseURL string) *tokenManager {
	return &tokenManager{
		clientID: clientID,
		apiKey:   apiKey,
		loginURL: baseURL + loginPath,
		now:      time.Now,
	}
}

// String implements fmt.Stringer with credentials redacted.
func (t *tokenManager) String() string {
	return fmt.Sprintf("tokenManager{clientID:%q apiKey:[REDACTED] loginURL:%q}",
		t.clientID, t.loginURL)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (t *tokenManager) GoString() string { return t.String() }

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// get returns a fresh bearer token, logging in when the cached one is
// missing or within the refresh leeway of expiring. Errors are either
// *ConnectionError (transport) or *Error (HTTP status / malformed body),
// so the caller's retry loop can classify them.
func (t *tokenManager) get(ctx context.Context, httpClient *http.Client) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" && t.now().Before(t.expiresAt.Add(-tokenRefreshLeeway)) {
		return t.token, nil
	}
	return t.login(ctx, httpClient)
}

// login must be called with t.mu held.
func (t *tokenManager) login(ctx context.Context, httpClient *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.loginURL, nil)
	if err != nil {
		return "", &ConnectionError{Message: "building login request", Err: err}
	}
	req.Header.Set("x-client-id", t.clientID)
	req.Header.Set("x-api-key", t.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", &ConnectionError{Message: "login request failed", Err: err}
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close of response body

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return "", &ConnectionError{Message: "reading login response", Err: err}
	}
	if resp.StatusCode >= 400 {
		return "", errorFromResponse(resp, body)
	}
	var parsed loginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", &Error{
			StatusCode: resp.StatusCode,
			RequestID:  resp.Header.Get("x-request-id"),
			Message: fmt.Sprintf(
				"login returned a %d response with an unparseable body (is a proxy intercepting the request?)",
				resp.StatusCode),
		}
	}
	if parsed.Token == "" {
		return "", &Error{
			StatusCode: resp.StatusCode,
			RequestID:  resp.Header.Get("x-request-id"),
			Message:    "login succeeded but no token was returned",
		}
	}
	t.token = parsed.Token
	t.expiresAt = parseExpiresAt(parsed.ExpiresAt, t.now())
	return t.token, nil
}

// invalidate clears the cached token so the next request logs in again.
func (t *tokenManager) invalidate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.token = ""
	t.expiresAt = time.Time{}
}
