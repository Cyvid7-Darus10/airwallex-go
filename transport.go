package airwallex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Version is the SDK release, sent in the User-Agent header.
const Version = "0.1.0"

// parseRetryAfter parses a Retry-After header in either delta-seconds or
// HTTP-date (RFC 7231) form. It returns false when the value is unusable.
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		return max(0, time.Duration(seconds*float64(time.Second))), true
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(0, at.Sub(now)), true
	}
	return 0, false
}

// retryDelay computes full-jitter exponential backoff, honouring a
// Retry-After header when the server sent one.
func retryDelay(attempt int, resp *http.Response, now time.Time) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), now); ok {
			return delay
		}
	}
	ceiling := initialRetryDelay << uint(attempt) //nolint:gosec // attempt is small
	if ceiling > maxRetryDelay || ceiling <= 0 {
		ceiling = maxRetryDelay
	}
	return time.Duration(rand.Float64() * float64(ceiling))
}

// sleepCtx waits for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// do sends an authenticated request with automatic retries and decodes the
// JSON response into out (which may be nil to discard the body).
//
// Retry policy — every branch shares one attempt budget of maxRetries:
//   - login transport errors and login 408/429/5xx are retried;
//   - request transport errors are retried;
//   - 408/429/5xx responses are retried with full-jitter backoff,
//     honouring Retry-After (seconds or HTTP-date);
//   - a single 401 triggers one token refresh + re-send without consuming
//     the retry budget; a second 401 surfaces as *Error;
//   - 409 and other 4xx are never retried.
//
// The request body bytes are marshalled once and re-sent verbatim, so any
// request_id inside is identical across retries (Airwallex idempotency).
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	requestURL := c.config.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("airwallex: encoding request body: %w", err)
		}
		bodyBytes = encoded
	}

	authRetried := false
	attempt := 0
	for {
		token, err := c.tokens.get(ctx, c.httpClient)
		if err != nil {
			// A transient auth-endpoint outage gets the same retry budget
			// as any other endpoint.
			if classifyForRetry(err) && attempt < c.config.maxRetries {
				if sleepErr := sleepCtx(ctx, retryDelay(attempt, nil, time.Now())); sleepErr != nil {
					return &ConnectionError{Message: "request cancelled while backing off", Err: sleepErr}
				}
				attempt++
				continue
			}
			return err
		}

		req, err := c.newRequest(ctx, method, requestURL, bodyBytes, token)
		if err != nil {
			return err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < c.config.maxRetries {
				if sleepErr := sleepCtx(ctx, retryDelay(attempt, nil, time.Now())); sleepErr != nil {
					return &ConnectionError{Message: "request cancelled while backing off", Err: sleepErr}
				}
				attempt++
				continue
			}
			return &ConnectionError{
				Message: fmt.Sprintf("request failed after %d attempt(s)", attempt+1),
				Err:     err,
			}
		}

		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		closeErr := resp.Body.Close()
		if readErr == nil {
			readErr = closeErr
		}
		if readErr != nil {
			if attempt < c.config.maxRetries {
				if sleepErr := sleepCtx(ctx, retryDelay(attempt, nil, time.Now())); sleepErr != nil {
					return &ConnectionError{Message: "request cancelled while backing off", Err: sleepErr}
				}
				attempt++
				continue
			}
			return &ConnectionError{Message: "reading response body", Err: readErr}
		}

		if resp.StatusCode == http.StatusUnauthorized && !authRetried {
			c.tokens.invalidate()
			authRetried = true
			continue
		}
		if isRetryableStatus(resp.StatusCode) && attempt < c.config.maxRetries {
			if sleepErr := sleepCtx(ctx, retryDelay(attempt, resp, time.Now())); sleepErr != nil {
				return &ConnectionError{Message: "request cancelled while backing off", Err: sleepErr}
			}
			attempt++
			continue
		}
		if resp.StatusCode >= 400 {
			return errorFromResponse(resp, respBody)
		}
		return decodeResponse(resp, respBody, out)
	}
}

// maxResponseBytes caps how much of a response is read into memory (64 MiB —
// far above any real Airwallex payload, guards against a misbehaving proxy).
const maxResponseBytes int64 = 64 << 20

// classifyForRetry reports whether an error from the token manager is
// transient. Login retries use plain backoff; Retry-After is honoured only
// for data-endpoint retries, where the response headers are at hand.
func classifyForRetry(err error) bool {
	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return true
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}
	return false
}

func (c *Client) newRequest(ctx context.Context, method, requestURL string, body []byte, token string) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, &ConnectionError{Message: "building request", Err: err}
	}
	req.Header.Set("User-Agent", userAgentPrefix+Version)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.config.apiVersion != "" {
		req.Header.Set("x-api-version", c.config.apiVersion)
	}
	if c.config.onBehalfOf != "" {
		req.Header.Set("x-on-behalf-of", c.config.onBehalfOf)
	}
	return req, nil
}

// decodeResponse unmarshals a 2xx body into out. Non-JSON bodies (e.g. HTML
// from an intercepting proxy) produce a typed *Error, never a raw
// json.SyntaxError. When out embeds APIResource, the raw body is preserved
// so no response data is ever lost.
func decodeResponse(resp *http.Response, body []byte, out any) error {
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &Error{
			StatusCode: resp.StatusCode,
			RequestID:  resp.Header.Get("x-request-id"),
			Message: fmt.Sprintf(
				"the API returned a %d response with an unparseable body (content-type: %s)",
				resp.StatusCode, resp.Header.Get("Content-Type")),
		}
	}
	if holder, ok := out.(rawCapturer); ok {
		holder.captureRaw(body)
	}
	return nil
}
