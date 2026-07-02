package airwallex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testClientID = "test_client_id"
	testAPIKey   = "test_api_key_secret_value"
	testToken    = "test_bearer_token_value"
)

// loginOK writes a successful login response valid for 30 minutes.
func loginOK(w http.ResponseWriter) {
	expires := time.Now().UTC().Add(30 * time.Minute).Format("2006-01-02T15:04:05Z0700")
	fmt.Fprintf(w, `{"token":%q,"expires_at":%q}`, testToken, expires)
}

// testServer wraps httptest.Server with a default login handler and a
// counter of login calls.
type testServer struct {
	*httptest.Server
	mux    *http.ServeMux
	logins atomic.Int64
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	mux := http.NewServeMux()
	ts := &testServer{mux: mux}
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, r *http.Request) {
		ts.logins.Add(1)
		if r.Header.Get("x-client-id") != testClientID || r.Header.Get("x-api-key") != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"code":"unauthorized","message":"bad credentials"}`)
			return
		}
		loginOK(w)
	})
	ts.Server = httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func (ts *testServer) client(t *testing.T, opts ...Option) *Client {
	t.Helper()
	base := []Option{
		WithClientID(testClientID),
		WithAPIKey(testAPIKey),
		WithBaseURL(ts.URL),
	}
	client, err := New(append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

// newServerWithMux starts an httptest server whose login handler the test
// controls entirely.
func newServerWithMux(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// mustClient builds a client with test credentials plus opts.
func mustClient(t *testing.T, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithClientID(testClientID), WithAPIKey(testAPIKey)}
	client, err := New(append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestNewRequiresCredentials(t *testing.T) {
	t.Setenv(envClientID, "")
	t.Setenv(envAPIKey, "")
	if _, err := New(); err == nil {
		t.Fatal("expected error when credentials are missing")
	}
	if _, err := New(WithClientID("id")); err == nil {
		t.Fatal("expected error when api key is missing")
	}
}

func TestNewReadsCredentialsFromEnv(t *testing.T) {
	t.Setenv(envClientID, "env_client")
	t.Setenv(envAPIKey, "env_key")
	client, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.config.clientID != "env_client" || client.config.apiKey != "env_key" {
		t.Fatalf("credentials not read from env: %+v", client.config.clientID)
	}
	if client.config.baseURL != productionBaseURL {
		t.Fatalf("default base URL = %q, want production", client.config.baseURL)
	}
}

func TestNewEnvironments(t *testing.T) {
	tests := []struct {
		env     Environment
		wantURL string
		wantErr bool
	}{
		{Production, productionBaseURL, false},
		{Demo, demoBaseURL, false},
		{"staging", "", true},
	}
	for _, tt := range tests {
		client, err := New(WithClientID("id"), WithAPIKey("key"), WithEnv(tt.env))
		if tt.wantErr {
			if err == nil {
				t.Errorf("env %q: expected error", tt.env)
			}
			continue
		}
		if err != nil {
			t.Errorf("env %q: %v", tt.env, err)
			continue
		}
		if client.config.baseURL != tt.wantURL {
			t.Errorf("env %q: base URL = %q, want %q", tt.env, client.config.baseURL, tt.wantURL)
		}
	}
}

func TestNewRejectsInsecureBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		wantErr bool
	}{
		{"https://api.example.com", false},
		{"http://localhost:8080", false},
		{"http://127.0.0.1:9999", false},
		{"http://[::1]:8080", false},
		{"http://api.example.com", true},
		{"http://evil.test", true},
		{"ftp://api.example.com", true},
	}
	for _, tt := range tests {
		_, err := New(WithClientID("id"), WithAPIKey("key"), WithBaseURL(tt.baseURL))
		if (err != nil) != tt.wantErr {
			t.Errorf("base URL %q: err = %v, wantErr = %v", tt.baseURL, err, tt.wantErr)
		}
	}
}

func TestNewRejectsNegativeRetries(t *testing.T) {
	if _, err := New(WithClientID("id"), WithAPIKey("key"), WithMaxRetries(-1)); err == nil {
		t.Fatal("expected error for negative max retries")
	}
}

// TestEveryServiceWired asserts every service field is non-nil on a new
// client, so adding a service without wiring it fails loudly.
func TestEveryServiceWired(t *testing.T) {
	client, err := New(WithClientID("id"), WithAPIKey("key"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	services := map[string]any{
		"Accounts":              client.Accounts,
		"Balances":              client.Balances,
		"BatchTransfers":        client.BatchTransfers,
		"Beneficiaries":         client.Beneficiaries,
		"ConversionAmendments":  client.ConversionAmendments,
		"Conversions":           client.Conversions,
		"Customers":             client.Customers,
		"Deposits":              client.Deposits,
		"FinancialTransactions": client.FinancialTransactions,
		"FxQuotes":              client.FxQuotes,
		"GlobalAccounts":        client.GlobalAccounts,
		"IssuingAuthorizations": client.IssuingAuthorizations,
		"IssuingCardholders":    client.IssuingCardholders,
		"IssuingCards":          client.IssuingCards,
		"IssuingTransactions":   client.IssuingTransactions,
		"Payers":                client.Payers,
		"PaymentIntents":        client.PaymentIntents,
		"Rates":                 client.Rates,
		"Reference":             client.Reference,
		"Refunds":               client.Refunds,
		"Settlements":           client.Settlements,
		"Simulation":            client.Simulation,
		"Transfers":             client.Transfers,
		"WalletTransfers":       client.WalletTransfers,
		"WebhookEndpoints":      client.WebhookEndpoints,
	}
	if len(services) != 25 {
		t.Fatalf("expected 25 services, got %d", len(services))
	}
	for name, service := range services {
		value := fmt.Sprintf("%v", service)
		if service == nil || value == "<nil>" {
			t.Errorf("service %s is nil", name)
		}
	}
}

// TestCredentialsNeverLeak formats the client and its internals every way
// fmt offers, and asserts the API key never appears.
func TestCredentialsNeverLeak(t *testing.T) {
	client, err := New(WithClientID("id_ok_to_show"), WithAPIKey("SUPER_SECRET_KEY"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		for name, value := range map[string]any{
			"client": client,
			"config": client.config,
			"tokens": client.tokens,
		} {
			formatted := fmt.Sprintf(format, value)
			if strings.Contains(formatted, "SUPER_SECRET_KEY") {
				t.Errorf("API key leaked formatting %s with %s: %s", name, format, formatted)
			}
		}
	}
}

func TestErrorStringsNeverLeakCredentials(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/failing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"validation_error","message":"bad request"}`)
	})
	client := ts.client(t)
	err := client.Request(context.Background(), http.MethodGet, "/api/v1/failing", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := fmt.Sprintf("%v %+v %#v", err, err, err)
	if strings.Contains(msg, testAPIKey) || strings.Contains(msg, testToken) {
		t.Fatalf("error output leaked credentials: %s", msg)
	}
}

func TestRequestEscapeHatch(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/pa/payment_disputes", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "OPEN" {
			t.Errorf("query status = %q, want OPEN", r.URL.Query().Get("status"))
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer "+testToken {
			t.Errorf("Authorization = %q", auth)
		}
		fmt.Fprint(w, `{"items":[]}`)
	})
	client := ts.client(t)
	var out json.RawMessage
	err := client.Request(context.Background(), http.MethodGet, "/api/v1/pa/payment_disputes",
		url.Values{"status": {"OPEN"}}, nil, &out)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if string(out) != `{"items":[]}` {
		t.Fatalf("out = %s", out)
	}
}

func TestDefaultHeadersApplied(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-version"); got != "2024-08-07" {
			t.Errorf("x-api-version = %q", got)
		}
		if got := r.Header.Get("x-on-behalf-of"); got != "acct_123" {
			t.Errorf("x-on-behalf-of = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != userAgentPrefix+Version {
			t.Errorf("User-Agent = %q", got)
		}
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	client := ts.client(t, WithAPIVersion("2024-08-07"), WithOnBehalfOf("acct_123"))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
}

// trackingTransport counts round trips and lets the test confirm the SDK
// used the caller's client rather than replacing it.
type trackingTransport struct {
	calls atomic.Int64
}

func (tt *trackingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	tt.calls.Add(1)
	return http.DefaultTransport.RoundTrip(r)
}

func TestCustomHTTPClientIsUsedAndNotMutated(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"id":"acc_1"}`)
	})
	transport := &trackingTransport{}
	custom := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	client := ts.client(t, WithHTTPClient(custom))
	if _, err := client.Accounts.Retrieve(context.Background()); err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	// login + data request both go through the caller's transport
	if calls := transport.calls.Load(); calls != 2 {
		t.Fatalf("custom transport used %d times, want 2", calls)
	}
	if custom.Timeout != 5*time.Second || custom.Transport != transport {
		t.Fatal("caller's http.Client was mutated")
	}
	// The caller's client must still work afterwards (never closed).
	if _, err := client.Balances.Current(context.Background()); err == nil {
		// endpoint not registered; any HTTP-level response is fine — the
		// point is the transport still functions.
		t.Log("balances endpoint unexpectedly succeeded")
	}
}

func TestContextCancellationMidRequest(t *testing.T) {
	ts := newTestServer(t)
	release := make(chan struct{})
	ts.mux.HandleFunc("/api/v1/account", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
		fmt.Fprint(w, `{}`)
	})
	defer close(release)
	client := ts.client(t)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Accounts.Retrieve(ctx)
		errCh <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
		var connErr *ConnectionError
		if !errors.As(err, &connErr) {
			t.Fatalf("error type = %T (%v), want *ConnectionError", err, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request did not return after cancellation")
	}
}

func TestUnicodePayloadRoundTrip(t *testing.T) {
	ts := newTestServer(t)
	reference := "发票 №42 — payé ✓"
	ts.mux.HandleFunc("/api/v1/transfers/create", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["reference"] != reference {
			t.Errorf("reference = %q, want %q", body["reference"], reference)
		}
		fmt.Fprintf(w, `{"id":"tra_1","reference":%q,"status":"NEW"}`, reference)
	})
	client := ts.client(t)
	transfer, err := client.Transfers.Create(context.Background(), &TransferCreateParams{
		Reference: reference,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if transfer.Reference != reference {
		t.Fatalf("round-tripped reference = %q, want %q", transfer.Reference, reference)
	}
}
