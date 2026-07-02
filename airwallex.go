// Package airwallex is an unofficial Go SDK for the Airwallex API —
// payouts, FX, balances, global accounts, payment acceptance, issuing,
// and webhooks.
//
//	client, err := airwallex.New(
//	    airwallex.WithClientID("..."), // or AIRWALLEX_CLIENT_ID
//	    airwallex.WithAPIKey("..."),   // or AIRWALLEX_API_KEY
//	    airwallex.WithEnv(airwallex.Demo),
//	)
//	balances, err := client.Balances.Current(ctx)
//
// Authentication happens lazily on the first request; the bearer token is
// cached and refreshed automatically before it expires. Transient failures
// (408/429/5xx/network) are retried with full-jitter exponential backoff,
// and money-moving creates carry an auto-generated request_id so retries
// are idempotent.
//
// This library is not affiliated with, endorsed by, or supported by
// Airwallex Pty Ltd.
package airwallex

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Client is the Airwallex API client. Create one with New and share it —
// it is safe for concurrent use.
type Client struct {
	config     *config
	httpClient *http.Client
	tokens     *tokenManager

	// Accounts retrieves details of your own Airwallex account.
	Accounts *AccountsService
	// Balances reports current and historical wallet balances.
	Balances *BalancesService
	// BatchTransfers manages batches of payouts.
	BatchTransfers *BatchTransfersService
	// Beneficiaries manages payout recipients.
	Beneficiaries *BeneficiariesService
	// ConversionAmendments amends or cancels existing conversions.
	ConversionAmendments *ConversionAmendmentsService
	// Conversions books FX conversions between wallet currencies.
	Conversions *ConversionsService
	// Customers manages payment-acceptance shoppers.
	Customers *CustomersService
	// Deposits lists deposits received into the wallet.
	Deposits *DepositsService
	// FinancialTransactions lists payment-acceptance ledger activity.
	FinancialTransactions *FinancialTransactionsService
	// FxQuotes creates lockable FX quotes.
	FxQuotes *FxQuotesService
	// GlobalAccounts manages local currency accounts for collecting funds.
	GlobalAccounts *GlobalAccountsService
	// IssuingAuthorizations lists card authorizations.
	IssuingAuthorizations *IssuingAuthorizationsService
	// IssuingCardholders manages people cards can be issued to.
	IssuingCardholders *IssuingCardholdersService
	// IssuingCards manages issued cards.
	IssuingCards *IssuingCardsService
	// IssuingTransactions lists card transactions.
	IssuingTransactions *IssuingTransactionsService
	// Payers manages the payers money is sent on behalf of.
	Payers *PayersService
	// PaymentIntents collects payments from shoppers.
	PaymentIntents *PaymentIntentsService
	// Rates fetches indicative FX rates.
	Rates *RatesService
	// Reference exposes static reference data.
	Reference *ReferenceService
	// Refunds refunds collected payments.
	Refunds *RefundsService
	// Settlements lists payment-acceptance settlements.
	Settlements *SettlementsService
	// Simulation drives demo-environment state transitions (sandbox only).
	Simulation *SimulationService
	// Transfers creates and manages payouts to beneficiaries.
	Transfers *TransfersService
	// WalletTransfers moves money between Airwallex wallets.
	WalletTransfers *WalletTransfersService
	// WebhookEndpoints manages webhook subscriptions.
	WebhookEndpoints *WebhookEndpointsService
}

// New creates a Client. Credentials default to the AIRWALLEX_CLIENT_ID and
// AIRWALLEX_API_KEY environment variables, and the environment defaults to
// Production.
func New(opts ...Option) (*Client, error) {
	cfg, err := newConfig(opts)
	if err != nil {
		return nil, err
	}
	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}
	c := &Client{
		config:     cfg,
		httpClient: httpClient,
		tokens:     newTokenManager(cfg.clientID, cfg.apiKey, cfg.baseURL),
	}
	c.Accounts = &AccountsService{client: c}
	c.Balances = &BalancesService{client: c}
	c.BatchTransfers = &BatchTransfersService{client: c}
	c.Beneficiaries = &BeneficiariesService{client: c}
	c.ConversionAmendments = &ConversionAmendmentsService{client: c}
	c.Conversions = &ConversionsService{client: c}
	c.Customers = &CustomersService{client: c}
	c.Deposits = &DepositsService{client: c}
	c.FinancialTransactions = &FinancialTransactionsService{client: c}
	c.FxQuotes = &FxQuotesService{client: c}
	c.GlobalAccounts = &GlobalAccountsService{client: c}
	c.IssuingAuthorizations = &IssuingAuthorizationsService{client: c}
	c.IssuingCardholders = &IssuingCardholdersService{client: c}
	c.IssuingCards = &IssuingCardsService{client: c}
	c.IssuingTransactions = &IssuingTransactionsService{client: c}
	c.Payers = &PayersService{client: c}
	c.PaymentIntents = &PaymentIntentsService{client: c}
	c.Rates = &RatesService{client: c}
	c.Reference = &ReferenceService{client: c}
	c.Refunds = &RefundsService{client: c}
	c.Settlements = &SettlementsService{client: c}
	c.Simulation = &SimulationService{client: c}
	c.Transfers = &TransfersService{client: c}
	c.WalletTransfers = &WalletTransfersService{client: c}
	c.WebhookEndpoints = &WebhookEndpointsService{client: c}
	return c, nil
}

// Request calls any Airwallex endpoint, including ones this SDK has no
// typed wrapper for yet. Authentication, retries, and error mapping still
// apply. body is JSON-encoded when non-nil; the response is decoded into
// out when non-nil.
//
//	var disputes json.RawMessage
//	err := client.Request(ctx, "GET", "/api/v1/pa/payment_disputes",
//	    url.Values{"status": {"OPEN"}}, nil, &disputes)
func (c *Client) Request(ctx context.Context, method, path string, params url.Values, body, out any) error {
	return c.do(ctx, method, path, params, body, out)
}

// String implements fmt.Stringer with credentials redacted.
func (c *Client) String() string {
	return fmt.Sprintf("airwallex.Client{environment:%q baseURL:%q}",
		c.config.environment, c.config.baseURL)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (c *Client) GoString() string { return c.String() }

// get issues an authenticated GET and decodes the response into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out)
}

// post issues an authenticated POST with a JSON body and decodes the
// response into out. body and out may be nil.
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out)
}
