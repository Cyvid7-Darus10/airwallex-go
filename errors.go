package airwallex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"
)

// Error is an error response from the Airwallex API.
//
// Use errors.As to inspect it:
//
//	var apiErr *airwallex.Error
//	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
//	    ...
//	}
type Error struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Code is the Airwallex machine-readable error code (e.g. "validation_error").
	Code string
	// Source is the field or parameter the error refers to, when provided.
	Source string
	// RequestID is the Airwallex request id from the x-request-id header —
	// include it when contacting Airwallex support.
	RequestID string
	// Message is the human-readable error description.
	Message string
	// Raw is the full error response body. Validation failures carry an
	// "errors" object here with per-field detail beyond Message.
	Raw json.RawMessage
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("airwallex: [%d] %s", e.StatusCode, e.Message)
	if e.Code != "" {
		msg += " code=" + e.Code
	}
	if e.Source != "" {
		msg += " source=" + e.Source
	}
	if e.RequestID != "" {
		msg += " request_id=" + e.RequestID
	}
	return msg
}

// IsRetryable reports whether the SDK considers this status transient
// (408, 429, or 5xx). 409 business conflicts are never retryable.
func (e *Error) IsRetryable() bool {
	return isRetryableStatus(e.StatusCode)
}

// ConnectionError means the request never received a valid HTTP response
// (network failure, timeout, cancelled context). It wraps the underlying
// transport error.
type ConnectionError struct {
	// Message describes the failed operation.
	Message string
	// Err is the underlying transport error.
	Err error
}

func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("airwallex: %s: %v", e.Message, e.Err)
	}
	return "airwallex: " + e.Message
}

func (e *ConnectionError) Unwrap() error { return e.Err }

// errorBody is the JSON shape of an Airwallex error response.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Source  string `json:"source"`
	Error   string `json:"error"`
}

// errorFromResponse builds the typed *Error for a non-2xx response.
// body may be nil or non-JSON (e.g. an HTML page from a proxy).
func errorFromResponse(resp *http.Response, body []byte) *Error {
	apiErr := &Error{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("x-request-id"),
		Raw:        json.RawMessage(append([]byte(nil), body...)),
	}
	var parsed errorBody
	if len(body) > 0 && json.Unmarshal(body, &parsed) == nil {
		apiErr.Code = parsed.Code
		apiErr.Source = parsed.Source
		apiErr.Message = parsed.Message
		if apiErr.Message == "" {
			apiErr.Message = parsed.Error
		}
	}
	if apiErr.Message == "" {
		if len(body) > 0 {
			apiErr.Message = truncate(string(body), 200)
		} else {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
	}
	return apiErr
}

// truncate shortens s to at most n bytes, backing up to the nearest rune
// boundary so multi-byte UTF-8 characters are never split.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
