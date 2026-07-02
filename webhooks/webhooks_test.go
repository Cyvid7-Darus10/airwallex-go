package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

const secret = "whsec_test_secret"

var fixedNow = time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)

func sign(t *testing.T, secret, timestamp string, payload []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func withFixedNow(t *testing.T) {
	t.Helper()
	previous := timeNow
	timeNow = func() time.Time { return fixedNow }
	t.Cleanup(func() { timeNow = previous })
}

func freshTimestampSeconds() string {
	return strconv.FormatInt(fixedNow.Unix(), 10)
}

func freshTimestampMillis() string {
	return strconv.FormatInt(fixedNow.UnixMilli(), 10)
}

func TestVerifySignatureValid(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1","name":"transfer.settled"}`)
	for name, timestamp := range map[string]string{
		"seconds":      freshTimestampSeconds(),
		"milliseconds": freshTimestampMillis(),
	} {
		t.Run(name, func(t *testing.T) {
			signature := sign(t, secret, timestamp, payload)
			if err := VerifySignature(payload, timestamp, signature, secret); err != nil {
				t.Fatalf("VerifySignature: %v", err)
			}
		})
	}
}

func TestVerifySignatureTamperedPayload(t *testing.T) {
	withFixedNow(t)
	timestamp := freshTimestampSeconds()
	signature := sign(t, secret, timestamp, []byte(`{"amount":100}`))
	err := VerifySignature([]byte(`{"amount":1000000}`), timestamp, signature, secret)
	if !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestVerifySignatureWrongSecret(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1"}`)
	timestamp := freshTimestampSeconds()
	signature := sign(t, "a_different_secret", timestamp, payload)
	if err := VerifySignature(payload, timestamp, signature, secret); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestVerifySignatureTamperedTimestamp(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1"}`)
	signature := sign(t, secret, freshTimestampSeconds(), payload)
	other := strconv.FormatInt(fixedNow.Unix()+30, 10)
	if err := VerifySignature(payload, other, signature, secret); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestVerifySignatureStaleTimestampRejected(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1"}`)
	stale := strconv.FormatInt(fixedNow.Add(-10*time.Minute).Unix(), 10)
	signature := sign(t, secret, stale, payload)
	err := VerifySignature(payload, stale, signature, secret)
	if !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature (replay)", err)
	}
	// A negative tolerance skips the check entirely.
	if err := VerifySignatureWithTolerance(payload, stale, signature, secret, -1); err != nil {
		t.Fatalf("negative tolerance should skip replay check: %v", err)
	}
}

func TestVerifySignatureFutureTimestampRejected(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1"}`)
	future := strconv.FormatInt(fixedNow.Add(10*time.Minute).Unix(), 10)
	signature := sign(t, secret, future, payload)
	if err := VerifySignature(payload, future, signature, secret); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestVerifySignatureInvalidTimestamp(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{}`)
	err := VerifySignature(payload, "yesterday", sign(t, secret, "yesterday", payload), secret)
	if !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestVerifySignatureRequiresSecret(t *testing.T) {
	if err := VerifySignature([]byte(`{}`), "0", "sig", ""); err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestConstructEventParsesFields(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1","name":"transfer.settled","account_id":"acc_9",` +
		`"created_at":"2026-07-02T09:59:00+0000","data":{"object":{"id":"tra_1","status":"SETTLED"}},` +
		`"unmodelled":"kept"}`)
	timestamp := freshTimestampMillis()
	signature := sign(t, secret, timestamp, payload)
	event, err := ConstructEvent(payload, timestamp, signature, secret)
	if err != nil {
		t.Fatalf("ConstructEvent: %v", err)
	}
	if event.ID != "evt_1" || event.Name != "transfer.settled" || event.AccountID != "acc_9" {
		t.Fatalf("event = %+v", event)
	}
	if len(event.Data) == 0 {
		t.Fatal("event.Data empty")
	}
	if string(event.Raw) != string(payload) {
		t.Fatal("event.Raw does not preserve the payload")
	}
}

func TestConstructEventRejectsBadSignature(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`{"id":"evt_1"}`)
	timestamp := freshTimestampSeconds()
	if _, err := ConstructEvent(payload, timestamp, "deadbeef", secret); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestConstructEventRejectsNonJSON(t *testing.T) {
	withFixedNow(t)
	payload := []byte(`this is not json`)
	timestamp := freshTimestampSeconds()
	signature := sign(t, secret, timestamp, payload)
	if _, err := ConstructEvent(payload, timestamp, signature, secret); !errors.Is(err, ErrSignature) {
		t.Fatalf("err = %v, want ErrSignature", err)
	}
}

func TestConstructEventUnicodePayload(t *testing.T) {
	withFixedNow(t)
	payload := []byte(fmt.Sprintf(`{"id":"evt_1","name":"transfer.settled","data":{"reference":%q}}`,
		"发票 №42 — payé ✓"))
	timestamp := freshTimestampSeconds()
	signature := sign(t, secret, timestamp, payload)
	if _, err := ConstructEvent(payload, timestamp, signature, secret); err != nil {
		t.Fatalf("ConstructEvent: %v", err)
	}
}
