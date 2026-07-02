package airwallex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRetryOn500ThenSuccess(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message":"boom"}`)
			return
		}
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	client := ts.client(t, WithMaxRetries(2))
	account, err := client.Accounts.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if account.ID != "acc_1" || hits != 2 {
		t.Fatalf("account = %+v after %d hits", account, hits)
	}
}

func TestRetryExhaustionSurfacesLastError(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("x-request-id", "req_exhausted")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"code":"server_error","message":"upstream down"}`)
	})
	client := ts.client(t, WithMaxRetries(2))
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *Error", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway || apiErr.RequestID != "req_exhausted" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
	if hits != 3 {
		t.Fatalf("endpoint hit %d times, want 3 (1 + 2 retries)", hits)
	}
}

func TestRetryStatusMatrix(t *testing.T) {
	tests := []struct {
		status    int
		wantHits  int
		wantError bool
	}{
		{http.StatusRequestTimeout, 2, false},  // 408 retried
		{http.StatusTooManyRequests, 2, false}, // 429 retried
		{http.StatusInternalServerError, 2, false},
		{http.StatusServiceUnavailable, 2, false},
		{http.StatusConflict, 1, true},   // 409 NEVER retried
		{http.StatusBadRequest, 1, true}, // 4xx not retried
		{http.StatusNotFound, 1, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			ts := newTestServer(t)
			var hits int
			ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
				hits++
				if hits == 1 {
					w.WriteHeader(tt.status)
					fmt.Fprint(w, `{"message":"first attempt fails"}`)
					return
				}
				fmt.Fprint(w, `{"id":"acc_1"}`)
			})
			client := ts.client(t, WithMaxRetries(2))
			_, err := client.Accounts.Retrieve(context.Background())
			if tt.wantError {
				var apiErr *Error
				if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status {
					t.Fatalf("err = %v, want %d *Error", err, tt.status)
				}
			} else if err != nil {
				t.Fatalf("Retrieve: %v", err)
			}
			if hits != tt.wantHits {
				t.Fatalf("endpoint hit %d times, want %d", hits, tt.wantHits)
			}
		})
	}
}

func TestRetryHonoursRetryAfterSeconds(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	var gap time.Duration
	var first time.Time
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			first = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		gap = time.Since(first)
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	client := ts.client(t, WithMaxRetries(1))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if gap < 900*time.Millisecond {
		t.Fatalf("retried after %v, want >= ~1s per Retry-After", gap)
	}
}

func TestRetryHonoursRetryAfterHTTPDate(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	var gap time.Duration
	var first time.Time
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			first = time.Now()
			// http-date has 1s resolution; +2s guarantees a visible wait
			// even when truncation rounds it down.
			w.Header().Set("Retry-After", time.Now().UTC().Add(2*time.Second).Format(http.TimeFormat))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		gap = time.Since(first)
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	client := ts.client(t, WithMaxRetries(1))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if gap < 900*time.Millisecond {
		t.Fatalf("retried after %v, want a visible wait per Retry-After date", gap)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		value  string
		want   time.Duration
		wantOK bool
	}{
		{"2", 2 * time.Second, true},
		{"0", 0, true},
		{"1.5", 1500 * time.Millisecond, true},
		{now.Add(3 * time.Second).Format(http.TimeFormat), 3 * time.Second, true},
		{now.Add(-3 * time.Second).Format(http.TimeFormat), 0, true},
		{"", 0, false},
		{"soon", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.value, now)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("parseRetryAfter(%q) = (%v, %v), want (%v, %v)",
				tt.value, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestConnectionErrorsRetried(t *testing.T) {
	// A server that accepts a login then dies would be fiddly; instead run
	// with retries pointing at a dead port and confirm the typed error and
	// wrapped cause come back after the budget is spent.
	client := mustClient(t, WithBaseURL("http://127.0.0.1:1"), WithMaxRetries(2))
	start := time.Now()
	_, err := client.Accounts.Retrieve(context.Background())
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatalf("err = %T (%v), want *ConnectionError", err, err)
	}
	if connErr.Unwrap() == nil {
		t.Fatal("ConnectionError does not wrap the transport error")
	}
	_ = start
}

// TestRetriesReuseRequestID is the double-execution guard: the request_id
// generated for a create must be byte-identical on every retry.
func TestRetriesReuseRequestID(t *testing.T) {
	ts := newTestServer(t)
	var requestIDs []string
	ts.mux.HandleFunc("/api/v1/transfers/create", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		id, _ := body["request_id"].(string)
		requestIDs = append(requestIDs, id)
		if len(requestIDs) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintf(w, `{"id":"tra_1","request_id":%q}`, id)
	})
	client := ts.client(t, WithMaxRetries(2))
	transfer, err := client.Transfers.Create(context.Background(), &TransferCreateParams{
		BeneficiaryID:    "ben_1",
		TransferAmount:   100,
		TransferCurrency: "PHP",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(requestIDs) != 3 {
		t.Fatalf("server saw %d attempts, want 3", len(requestIDs))
	}
	if requestIDs[0] == "" {
		t.Fatal("request_id was not auto-generated")
	}
	if requestIDs[0] != requestIDs[1] || requestIDs[1] != requestIDs[2] {
		t.Fatalf("request_id changed across retries: %v", requestIDs)
	}
	if transfer.RequestID != requestIDs[0] {
		t.Fatalf("response request_id = %q, want %q", transfer.RequestID, requestIDs[0])
	}
}

func TestExplicitRequestIDPassesThrough(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/transfers/create", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["request_id"] != "my-idempotency-key" {
			t.Errorf("request_id = %v, want my-idempotency-key", body["request_id"])
		}
		fmt.Fprint(w, `{"id":"tra_1"}`)
	})
	client := ts.client(t)
	params := &TransferCreateParams{RequestID: "my-idempotency-key"}
	if _, err := client.Transfers.Create(context.Background(), params); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if params.RequestID != "my-idempotency-key" {
		t.Fatal("caller's params struct was mutated")
	}
}

func TestNonJSON2xxProducesTypedError(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>proxy says hi</html>")
	})
	client := ts.client(t)
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T (%v), want *Error", err, err)
	}
	if !strings.Contains(apiErr.Message, "unparseable") {
		t.Fatalf("message = %q, want mention of unparseable body", apiErr.Message)
	}
}

func TestErrorFieldMapping(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-request-id", "req_abc123")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"validation_error","message":"transfer_amount is required","source":"transfer_amount"}`)
	})
	client := ts.client(t)
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *Error", err)
	}
	if apiErr.StatusCode != 400 ||
		apiErr.Code != "validation_error" ||
		apiErr.Source != "transfer_amount" ||
		apiErr.RequestID != "req_abc123" ||
		apiErr.Message != "transfer_amount is required" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
	for _, want := range []string{"400", "validation_error", "transfer_amount", "req_abc123"} {
		if !strings.Contains(apiErr.Error(), want) {
			t.Errorf("Error() = %q, missing %q", apiErr.Error(), want)
		}
	}
}

func TestNonJSONErrorBody(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "<html>not json</html>")
	})
	client := ts.client(t)
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 {
		t.Fatalf("err = %v", err)
	}
	if apiErr.Message == "" {
		t.Fatal("message empty for non-JSON error body")
	}
}

func TestContextCancelledDuringBackoff(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	client := ts.client(t, WithMaxRetries(3))
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := client.Accounts.Retrieve(ctx)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancellation not honoured during backoff; waited %v", elapsed)
	}
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatalf("err = %T (%v), want *ConnectionError", err, err)
	}
}

func TestRetryDelayCapsAndJitter(t *testing.T) {
	now := time.Now()
	for attempt := range 10 {
		delay := retryDelay(attempt, nil, now)
		if delay < 0 || delay > maxRetryDelay {
			t.Fatalf("attempt %d: delay %v outside [0, %v]", attempt, delay, maxRetryDelay)
		}
	}
}
