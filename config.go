package airwallex

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Environment selects which Airwallex API host the client talks to.
type Environment string

const (
	// Production is the live Airwallex API (https://api.airwallex.com).
	Production Environment = "production"
	// Demo is the Airwallex sandbox (https://api-demo.airwallex.com).
	Demo Environment = "demo"
)

const (
	productionBaseURL = "https://api.airwallex.com"
	demoBaseURL       = "https://api-demo.airwallex.com"

	loginPath = "/api/v1/authentication/login"

	// Airwallex bearer tokens live ~30 minutes; refresh this many seconds
	// early so an in-flight request never carries a token that expires
	// mid-request.
	tokenRefreshLeeway = 60 * time.Second

	defaultTimeout          = 60 * time.Second
	defaultMaxRetries       = 2
	initialRetryDelay       = 500 * time.Millisecond
	maxRetryDelay           = 8 * time.Second
	envClientID             = "AIRWALLEX_CLIENT_ID"
	envAPIKey               = "AIRWALLEX_API_KEY" //nolint:gosec // env var name, not a credential
	userAgentPrefix         = "airwallex-go/"
	fallbackTokenTTL        = 30 * time.Minute
	maxErrorBodyBytes int64 = 1 << 20 // cap error bodies read into memory
)

// isRetryableStatus reports whether a status code should be retried.
// 409 is deliberately absent: Airwallex uses it for business conflicts
// (duplicate request_id, invalid state transitions) that must surface to
// the caller immediately, never be silently retried.
func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	}
	return status >= 500 && status <= 599
}

type config struct {
	clientID    string
	apiKey      string
	environment Environment
	baseURL     string
	apiVersion  string
	onBehalfOf  string
	timeout     time.Duration
	maxRetries  int
	httpClient  *http.Client
	logger      *slog.Logger
}

// String implements fmt.Stringer with the API key redacted, so accidental
// logging of the client never leaks credentials.
func (c *config) String() string {
	return fmt.Sprintf("config{clientID:%q apiKey:[REDACTED] environment:%q baseURL:%q}",
		c.clientID, c.environment, c.baseURL)
}

// GoString implements fmt.GoStringer (%#v) with the API key redacted.
func (c *config) GoString() string { return c.String() }

// Option configures a Client created by New.
type Option func(*config)

// WithClientID sets the Airwallex client id. Defaults to the
// AIRWALLEX_CLIENT_ID environment variable.
func WithClientID(clientID string) Option {
	return func(c *config) { c.clientID = clientID }
}

// WithAPIKey sets the Airwallex API key. Defaults to the
// AIRWALLEX_API_KEY environment variable.
func WithAPIKey(apiKey string) Option {
	return func(c *config) { c.apiKey = apiKey }
}

// WithEnv selects the production or demo (sandbox) environment.
// The default is Production.
func WithEnv(env Environment) Option {
	return func(c *config) { c.environment = env }
}

// WithBaseURL overrides the API host entirely (advanced; wins over WithEnv).
// The URL must use https; plain http is allowed only for loopback hosts.
func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithAPIVersion pins an x-api-version header (e.g. "2024-08-07") instead of
// your account's default API version.
func WithAPIVersion(version string) Option {
	return func(c *config) { c.apiVersion = version }
}

// WithOnBehalfOf acts on a connected account (sets x-on-behalf-of on every
// request).
func WithOnBehalfOf(accountID string) Option {
	return func(c *config) { c.onBehalfOf = accountID }
}

// WithTimeout sets the per-request timeout for the HTTP client the SDK
// constructs. It has no effect when WithHTTPClient supplies a custom client —
// configure the timeout on that client instead.
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) { c.timeout = timeout }
}

// WithMaxRetries sets how many times transient failures (408/429/5xx/network)
// are retried. The default is 2. Retries reuse the same request_id, so
// money-moving calls are never executed twice.
func WithMaxRetries(maxRetries int) Option {
	return func(c *config) { c.maxRetries = maxRetries }
}

// WithHTTPClient supplies a custom *http.Client (proxies, custom TLS, ...).
// The SDK applies the base URL and default headers per request and never
// mutates or closes the supplied client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *config) { c.httpClient = httpClient }
}

// WithLogger enables debug logging of request outcomes, retries, and token
// refreshes through the given structured logger. Only non-sensitive facts
// (method, path, status, attempt, request id, delay) are logged — never
// credentials, tokens, headers, or bodies. Logging is off by default.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

var insecureHosts = map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}

// validateBaseURL requires https so credentials are never sent in cleartext.
// Plain http is allowed only for loopback hosts (local mocks and tests).
func validateBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("airwallex: invalid base URL %q: %w", baseURL, err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && insecureHosts[parsed.Hostname()] {
		return nil
	}
	return fmt.Errorf(
		"airwallex: base URL must use https (got %q); plain http is only allowed for localhost",
		baseURL)
}

func newConfig(opts []Option) (*config, error) {
	cfg := &config{
		environment: Production,
		timeout:     defaultTimeout,
		maxRetries:  defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.clientID == "" {
		cfg.clientID = os.Getenv(envClientID)
	}
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv(envAPIKey)
	}
	if cfg.clientID == "" || cfg.apiKey == "" {
		return nil, fmt.Errorf(
			"airwallex: missing credentials: pass WithClientID/WithAPIKey or set the %s and %s environment variables",
			envClientID, envAPIKey)
	}
	if cfg.maxRetries < 0 {
		return nil, fmt.Errorf("airwallex: max retries must be >= 0, got %d", cfg.maxRetries)
	}
	if cfg.baseURL == "" {
		switch cfg.environment {
		case Production:
			cfg.baseURL = productionBaseURL
		case Demo:
			cfg.baseURL = demoBaseURL
		default:
			return nil, fmt.Errorf("airwallex: environment must be %q or %q, got %q",
				Production, Demo, cfg.environment)
		}
	}
	if err := validateBaseURL(cfg.baseURL); err != nil {
		return nil, err
	}
	return cfg, nil
}
