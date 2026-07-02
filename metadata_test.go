package airwallex

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestLastResponseOnRetrieve(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/transfers/tra_1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-request-id", "req_meta_1")
		fmt.Fprint(w, `{"id":"tra_1","status":"PAID"}`)
	})
	client := ts.client(t)
	transfer, err := client.Transfers.Retrieve(context.Background(), "tra_1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	meta := transfer.LastResponse
	if meta == nil {
		t.Fatal("LastResponse not captured")
	}
	if meta.StatusCode != 200 || meta.RequestID != "req_meta_1" {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.Header.Get("x-request-id") != "req_meta_1" {
		t.Fatalf("header not preserved: %v", meta.Header)
	}
}

func TestLastResponseOnPagesAndItems(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/transfers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-request-id", "req_page_1")
		fmt.Fprint(w, `{"has_more":false,"items":[{"id":"tra_1"}]}`)
	})
	client := ts.client(t)
	page, err := client.Transfers.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.LastResponse == nil || page.LastResponse.RequestID != "req_page_1" {
		t.Fatalf("page.LastResponse = %+v", page.LastResponse)
	}
	if page.Items[0].LastResponse == nil || page.Items[0].LastResponse.RequestID != "req_page_1" {
		t.Fatalf("item.LastResponse = %+v", page.Items[0].LastResponse)
	}
}

func TestLastResponseOnBalancesCurrent(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/balances/current", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-request-id", "req_bal_1")
		fmt.Fprint(w, `[{"currency":"USD","available_amount":10}]`)
	})
	client := ts.client(t)
	balances, err := client.Balances.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if len(balances) != 1 || balances[0].LastResponse == nil ||
		balances[0].LastResponse.RequestID != "req_bal_1" {
		t.Fatalf("balances = %+v", balances)
	}
}

func TestRequestWithHeaders(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/custom", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-version"); got != "2020-01-01" {
			t.Errorf("x-api-version = %q, want per-call override", got)
		}
		// Authorization must stay SDK-managed even if the caller tries to set it.
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("Authorization = %q, want SDK-managed bearer", got)
		}
		fmt.Fprint(w, `{}`)
	})
	client := ts.client(t, WithAPIVersion("2024-08-07"))
	err := client.RequestWithHeaders(context.Background(), http.MethodGet, "/api/v1/custom",
		nil, http.Header{
			"x-api-version": {"2020-01-01"},
			"Authorization": {"Bearer attacker-controlled"},
		}, nil, nil)
	if err != nil {
		t.Fatalf("RequestWithHeaders: %v", err)
	}
}

func TestWithLoggerEmitsRedactedDebugLogs(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("x-request-id", "req_log_1")
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := ts.client(t, WithLogger(logger), WithMaxRetries(1))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	logs := buf.String()
	for _, want := range []string{"request completed", "retrying after transient status", "status=503", "req_log_1"} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs missing %q:\n%s", want, logs)
		}
	}
	for _, secret := range []string{testAPIKey, testToken} {
		if strings.Contains(logs, secret) {
			t.Fatalf("logs leaked a credential:\n%s", logs)
		}
	}
}

func TestNoLoggerMeansNoOutput(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	client := ts.client(t)
	// Must not panic with a nil logger.
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
}
