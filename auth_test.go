package airwallex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func okAccount(ts *testServer) {
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
}

func TestTokenIsCachedAcrossRequests(t *testing.T) {
	ts := newTestServer(t)
	okAccount(ts)
	client := ts.client(t)
	ctx := context.Background()
	for range 3 {
		if _, err := client.Accounts.Retrieve(ctx); err != nil {
			t.Fatalf("Retrieve: %v", err)
		}
	}
	if logins := ts.logins.Load(); logins != 1 {
		t.Fatalf("logged in %d times for 3 requests, want 1", logins)
	}
}

func TestTokenRefreshedBeforeExpiry(t *testing.T) {
	ts := newTestServer(t)
	okAccount(ts)
	client := ts.client(t)
	ctx := context.Background()
	if _, err := client.Accounts.Retrieve(ctx); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	// Simulate time passing to within the refresh leeway of expiry.
	client.tokens.mu.Lock()
	client.tokens.expiresAt = time.Now().Add(tokenRefreshLeeway / 2)
	client.tokens.mu.Unlock()
	if _, err := client.Accounts.Retrieve(ctx); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if logins := ts.logins.Load(); logins != 2 {
		t.Fatalf("logged in %d times, want 2 (refresh before expiry)", logins)
	}
}

func TestSingleReloginOn401ThenFail(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":"unauthorized","message":"token expired"}`)
	})
	client := ts.client(t)
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want 401 *Error", err)
	}
	if hits != 2 {
		t.Fatalf("endpoint hit %d times, want exactly 2 (original + one re-login retry)", hits)
	}
	if logins := ts.logins.Load(); logins != 2 {
		t.Fatalf("logged in %d times, want 2", logins)
	}
}

func Test401RecoveredBySingleRelogin(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message":"token expired"}`)
			return
		}
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	client := ts.client(t)
	account, err := client.Accounts.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if account.ID != "acc_1" || hits != 2 {
		t.Fatalf("account = %+v, hits = %d", account, hits)
	}
}

func TestLoginFailureSurfacesTypedError(t *testing.T) {
	ts := newTestServer(t)
	okAccount(ts)
	client, err := New(
		WithClientID("wrong"), WithAPIKey("also wrong"), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want 401 *Error", err)
	}
}

func TestNonJSONLoginBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"html", "<html>gateway error</html>"},
		{"no token", `{"expires_at":"2030-01-01T00:00:00+0000"}`},
		{"empty token", `{"token":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, tt.body)
			})
			server := newServerWithMux(t, mux)
			client := mustClient(t, WithBaseURL(server.URL))
			_, err := client.Accounts.Retrieve(context.Background())
			var apiErr *Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %T (%v), want *Error", err, err)
			}
		})
	}
}

func TestLoginRetriedOnServerError(t *testing.T) {
	mux := http.NewServeMux()
	var loginHits int
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		loginHits++
		if loginHits == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		loginOK(w)
	})
	mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	server := newServerWithMux(t, mux)
	client := mustClient(t, WithBaseURL(server.URL), WithMaxRetries(2))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if loginHits != 2 {
		t.Fatalf("login hit %d times, want 2 (5xx retried)", loginHits)
	}
}

func TestLoginRetriesExhaust(t *testing.T) {
	mux := http.NewServeMux()
	var loginHits int
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		loginHits++
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	server := newServerWithMux(t, mux)
	client := mustClient(t, WithBaseURL(server.URL), WithMaxRetries(1))
	_, err := client.Accounts.Retrieve(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("err = %v, want 503 *Error", err)
	}
	if loginHits != 2 {
		t.Fatalf("login hit %d times, want 2 (1 + 1 retry)", loginHits)
	}
}

func TestLoginNetworkErrorRetried(t *testing.T) {
	// Point at a closed port: every attempt is a connection error.
	client := mustClient(t, WithBaseURL("http://127.0.0.1:1"), WithMaxRetries(1))
	_, err := client.Accounts.Retrieve(context.Background())
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatalf("err = %T (%v), want *ConnectionError", err, err)
	}
}

func TestParseExpiresAt(t *testing.T) {
	now := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		raw  string
		want time.Time
	}{
		{"2026-01-01T00:00:00+0000", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2026-01-01T00:00:00Z", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2026-01-01T08:00:00+0800", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2026-01-01T00:00:00.123+0000", time.Date(2026, 1, 1, 0, 0, 0, 123000000, time.UTC)},
		{"", now.Add(fallbackTokenTTL)},
		{"not-a-date", now.Add(fallbackTokenTTL)},
	}
	for _, tt := range tests {
		got := parseExpiresAt(tt.raw, now)
		if !got.Equal(tt.want) {
			t.Errorf("parseExpiresAt(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestConcurrentRequestsLoginOnce(t *testing.T) {
	ts := newTestServer(t)
	okAccount(ts)
	client := ts.client(t)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
				t.Errorf("Retrieve: %v", err)
			}
		}()
	}
	wg.Wait()
	if logins := ts.logins.Load(); logins != 1 {
		t.Fatalf("10 concurrent requests performed %d logins, want 1", logins)
	}
}
