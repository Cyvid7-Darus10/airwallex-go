package airwallex

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestConnectionErrorFormatting(t *testing.T) {
	cause := errors.New("dial tcp: connection refused")
	err := &ConnectionError{Message: "request failed", Err: cause}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("Unwrap chain broken")
	}
	bare := &ConnectionError{Message: "no cause"}
	if bare.Error() != "airwallex: no cause" {
		t.Fatalf("Error() = %q", bare.Error())
	}
}

func TestErrorIsRetryable(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{408, true}, {429, true}, {500, true}, {503, true},
		{409, false}, {400, false}, {401, false}, {404, false},
	}
	for _, tt := range tests {
		err := &Error{StatusCode: tt.status}
		if err.IsRetryable() != tt.want {
			t.Errorf("IsRetryable(%d) = %v, want %v", tt.status, err.IsRetryable(), tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 3); got != "hel" {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Fatalf("truncate = %q", got)
	}
	// Never split a multi-byte rune: "héllo" is h(1) é(2) l l o — cutting at
	// byte 2 lands inside é and must back up to "h".
	if got := truncate("héllo", 2); got != "h" {
		t.Fatalf("truncate mid-rune = %q, want %q", got, "h")
	}
	if got := truncate("发票发票", 5); !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
}

func TestWithTimeoutAppliesToOwnedClient(t *testing.T) {
	client := mustClient(t, WithTimeout(7*time.Second))
	if client.httpClient.Timeout != 7*time.Second {
		t.Fatalf("timeout = %v, want 7s", client.httpClient.Timeout)
	}
}

func TestBodyMapRejectsNonObjectParams(t *testing.T) {
	if _, err := bodyMap([]string{"not", "an", "object"}); err == nil {
		t.Fatal("expected error for non-object params")
	}
	if _, err := bodyMap(func() {}); err == nil {
		t.Fatal("expected error for unmarshalable params")
	}
}

func TestBodyMapNilVariants(t *testing.T) {
	for _, params := range []any{nil, (*TransferCreateParams)(nil)} {
		body, err := bodyMap(params)
		if err != nil || len(body) != 0 {
			t.Fatalf("bodyMap(%v) = (%v, %v)", params, body, err)
		}
	}
}

func TestEncodeQueryVariants(t *testing.T) {
	type sample struct {
		Name    string  `json:"name,omitempty"`
		Amount  float64 `json:"amount,omitempty"`
		Whole   float64 `json:"whole,omitempty"`
		Flag    bool    `json:"flag,omitempty"`
		Skipped string  `json:"skipped,omitempty"`
	}
	values, err := encodeQuery(&sample{Name: "a b", Amount: 1.5, Whole: 3, Flag: true})
	if err != nil {
		t.Fatalf("encodeQuery: %v", err)
	}
	want := url.Values{"name": {"a b"}, "amount": {"1.5"}, "whole": {"3"}, "flag": {"true"}}
	for key, values2 := range want {
		if values.Get(key) != values2[0] {
			t.Errorf("%s = %q, want %q", key, values.Get(key), values2[0])
		}
	}
	if values.Has("skipped") {
		t.Error("zero value not omitted")
	}
	if _, err := encodeQuery("not an object"); err == nil {
		t.Fatal("expected error for non-object query params")
	}
	empty, err := encodeQuery(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("encodeQuery(nil) = (%v, %v)", empty, err)
	}
}

func TestQueryValueFallback(t *testing.T) {
	if got := queryValue(json.RawMessage(`{"nested":1}`)); got != `{"nested":1}` {
		t.Fatalf("queryValue = %q", got)
	}
}

func TestDecodeItemsError(t *testing.T) {
	if _, err := decodeItems[Transfer]([]json.RawMessage{json.RawMessage(`"not an object"`)}, nil); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestIdempotentBodyPreservesExistingID(t *testing.T) {
	body, err := idempotentBody(&TransferCreateParams{RequestID: "keep-me"})
	if err != nil {
		t.Fatalf("idempotentBody: %v", err)
	}
	if body["request_id"] != "keep-me" {
		t.Fatalf("request_id = %v", body["request_id"])
	}
	generated, err := idempotentBody(nil)
	if err != nil {
		t.Fatalf("idempotentBody(nil): %v", err)
	}
	if generated["request_id"] == "" {
		t.Fatal("request_id not generated for nil params")
	}
}

func TestSleepCtx(t *testing.T) {
	if err := sleepCtx(context.Background(), 0); err != nil {
		t.Fatalf("zero sleep: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(cancelled, time.Hour); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestClassifyForRetry(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{&ConnectionError{Message: "net"}, true},
		{&Error{StatusCode: 503}, true},
		{&Error{StatusCode: 401}, false},
		{errors.New("plain"), false},
	}
	for _, tt := range tests {
		if got := classifyForRetry(tt.err); got != tt.want {
			t.Errorf("classifyForRetry(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
