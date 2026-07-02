// Package webhooks verifies and parses incoming Airwallex webhook
// notifications.
//
// Airwallex signs every webhook with your endpoint's secret:
// x-signature = hex(HMAC-SHA256(secret, x-timestamp + raw_body)).
//
// Typical usage inside an HTTP handler:
//
//	event, err := webhooks.ConstructEvent(
//	    rawBody, // exactly as received — do not re-serialise the JSON
//	    r.Header.Get("x-timestamp"),
//	    r.Header.Get("x-signature"),
//	    secret,
//	)
//	if err != nil {
//	    w.WriteHeader(http.StatusBadRequest)
//	    return
//	}
//	if event.Name == "transfer.settled" { ... }
package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// DefaultTolerance is how far a webhook's timestamp may differ from the
// current time before it is rejected as a possible replay.
const DefaultTolerance = 5 * time.Minute

// ErrSignature is wrapped by every verification failure, so a single
// errors.Is check catches tampered payloads, wrong secrets, and stale
// timestamps alike.
var ErrSignature = errors.New("webhook signature verification failed")

// Event is a parsed webhook notification. Data holds the resource payload;
// its exact shape depends on Name — see the Airwallex event types
// documentation.
type Event struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	AccountID string          `json:"account_id"`
	Data      json.RawMessage `json:"data"`
	CreatedAt string          `json:"created_at"`

	// Raw is the full verified payload, for fields not modelled above.
	Raw json.RawMessage `json:"-"`
}

// timeNow is stubbed in tests.
var timeNow = time.Now

// VerifySignature checks a webhook payload's authenticity with
// DefaultTolerance replay protection. Pass the raw request body exactly as
// received — re-serialising the JSON changes the bytes and invalidates the
// signature. All failures wrap ErrSignature.
func VerifySignature(payload []byte, timestamp, signature, secret string) error {
	return VerifySignatureWithTolerance(payload, timestamp, signature, secret, DefaultTolerance)
}

// VerifySignatureWithTolerance is VerifySignature with a custom replay
// tolerance. A negative tolerance skips the timestamp check entirely
// (useful when replaying stored deliveries in tests).
func VerifySignatureWithTolerance(payload []byte, timestamp, signature, secret string, tolerance time.Duration) error {
	if secret == "" {
		return errors.New("webhooks: secret is required to verify signatures")
	}
	if tolerance >= 0 {
		if err := checkTimestamp(timestamp, tolerance); err != nil {
			return err
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("webhooks: signature does not match the payload: %w", ErrSignature)
	}
	return nil
}

func checkTimestamp(timestamp string, tolerance time.Duration) error {
	sentAt, err := strconv.ParseFloat(timestamp, 64)
	if err != nil {
		return fmt.Errorf("webhooks: invalid x-timestamp header %q: %w", timestamp, ErrSignature)
	}
	// Airwallex sends unix timestamps in milliseconds; tolerate seconds too.
	if sentAt > 1e12 {
		sentAt /= 1000
	}
	drift := timeNow().UTC().Sub(time.Unix(0, int64(sentAt*float64(time.Second))))
	if drift < 0 {
		drift = -drift
	}
	if drift > tolerance {
		return fmt.Errorf(
			"webhooks: timestamp is outside the allowed tolerance of %s; possible replay: %w",
			tolerance, ErrSignature)
	}
	return nil
}

// ConstructEvent verifies the signature (with DefaultTolerance replay
// protection) and returns the parsed Event.
func ConstructEvent(payload []byte, timestamp, signature, secret string) (Event, error) {
	return ConstructEventWithTolerance(payload, timestamp, signature, secret, DefaultTolerance)
}

// ConstructEventWithTolerance is ConstructEvent with a custom replay
// tolerance; a negative tolerance skips the timestamp check.
func ConstructEventWithTolerance(payload []byte, timestamp, signature, secret string, tolerance time.Duration) (Event, error) {
	if err := VerifySignatureWithTolerance(payload, timestamp, signature, secret, tolerance); err != nil {
		return Event{}, err
	}
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return Event{}, fmt.Errorf("webhooks: payload is not valid JSON: %w", ErrSignature)
	}
	event.Raw = json.RawMessage(append([]byte(nil), payload...))
	return event, nil
}
