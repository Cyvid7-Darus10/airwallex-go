# airwallex-go

**Unofficial** Go SDK for the [Airwallex API](https://www.airwallex.com/docs/api) — payouts, FX, balances, global accounts, beneficiaries, payment acceptance, issuing, and webhooks.

[![CI](https://github.com/Cyvid7-Darus10/airwallex-go/actions/workflows/ci.yml/badge.svg)](https://github.com/Cyvid7-Darus10/airwallex-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Cyvid7-Darus10/airwallex-go.svg)](https://pkg.go.dev/github.com/Cyvid7-Darus10/airwallex-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cyvid7-Darus10/airwallex-go)](https://goreportcard.com/report/github.com/Cyvid7-Darus10/airwallex-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status: Beta](https://img.shields.io/badge/status-beta-orange.svg)](#status)

> [!IMPORTANT]
> This is an **unofficial, community-maintained** library. It is **not** affiliated with, endorsed by, or supported by Airwallex Pty Ltd — "Airwallex" is their trademark, used here only to describe compatibility. The SDK is in **beta**: the public interface may change before v1.0, so pin your version. For vendor-supported tooling, use the [official Node.js SDK](https://www.npmjs.com/package/@airwallex/node-sdk).

Airwallex's only official server-side SDK is Node.js. This library brings the same developer experience to Go, mirroring the [airwallex-python](https://github.com/Cyvid7-Darus10/airwallex-python) SDK:

- **One idiomatic client** — services as fields, `context.Context` on every call, zero third-party dependencies (standard library only)
- **Automatic authentication** — token fetched on first use and refreshed before expiry; no manual login calls
- **Idempotent by default** — `request_id` is auto-generated for money-moving calls, so retries never double-pay
- **Automatic retries** with full-jitter exponential backoff on 408/429/5xx/network failures (honours `Retry-After` in both seconds and HTTP-date form; 409 business conflicts are never retried)
- **Typed responses** that are forward-compatible — every resource keeps the raw response JSON in `.Raw`, so fields from newer API versions are never lost
- **Auto-pagination** — walk every page with one `range` loop (Go 1.23 iterators)
- **Webhook signature verification** with constant-time comparison and replay protection
- **Typed errors** — `*airwallex.Error` carries the HTTP status, Airwallex error `code`, `source`, and `x-request-id`; transport failures are a distinct `*airwallex.ConnectionError`
- **Response metadata everywhere** — every resource and page exposes `LastResponse` (status, `x-request-id`, headers), like stripe-go
- **Opt-in structured logging** via `log/slog` (`WithLogger`) — request outcomes, retries, and token refreshes at debug level, with credentials never logged

## Installation

```bash
go get github.com/Cyvid7-Darus10/airwallex-go
```

Requires Go 1.23+. Releases follow [semantic versioning](#status); see the [changelog](CHANGELOG.md).

## Quickstart

Create API credentials in the Airwallex web app under **Developer → API keys**, then:

```go
import "github.com/Cyvid7-Darus10/airwallex-go"

client, err := airwallex.New(
    airwallex.WithClientID("your_client_id"), // or set AIRWALLEX_CLIENT_ID
    airwallex.WithAPIKey("your_api_key"),     // or set AIRWALLEX_API_KEY
    airwallex.WithEnv(airwallex.Demo),        // airwallex.Production is the default
)
if err != nil {
    log.Fatal(err)
}

// Current wallet balances
balances, err := client.Balances.Current(ctx)
for _, balance := range balances {
    fmt.Println(balance.Currency, balance.AvailableAmount)
}
```

### Send a payout

> Payouts use `/api/v1/transfers`, which requires API version 2024-01-31 or later. If your account default is older, pass `airwallex.WithAPIVersion("2024-01-31")` (or newer).

```go
transfer, err := client.Transfers.Create(ctx, &airwallex.TransferCreateParams{
    BeneficiaryID:    "ben_abc123",
    SourceCurrency:   "USD",
    TransferCurrency: "PHP",
    TransferAmount:   5000,
    TransferMethod:   "LOCAL",
    Reference:        "Invoice 42",
    Reason:           "professional_service_fees",
})
fmt.Println(transfer.ID, transfer.Status)
```

`RequestID` is generated for you (set it to control idempotency yourself). Airwallex will never execute the same `request_id` twice — including across the SDK's automatic retries.

### FX: quote and convert

```go
rate, err := client.Rates.Current(ctx, &airwallex.RateCurrentParams{
    BuyCurrency: "USD", SellCurrency: "SGD", BuyAmount: 1000,
})
fmt.Println(rate.Rate)

conversion, err := client.Conversions.Create(ctx, &airwallex.ConversionCreateParams{
    BuyCurrency:   "USD",
    SellCurrency:  "SGD",
    BuyAmount:     1000,
    TermAgreement: true,
    Reason:        "hedging",
})
```

### Accept a payment

```go
intent, err := client.PaymentIntents.Create(ctx, &airwallex.PaymentIntentCreateParams{
    Amount:          25.00,
    Currency:        "USD",
    MerchantOrderID: "order_42",
})
confirmed, err := client.PaymentIntents.Confirm(ctx, intent.ID, &airwallex.PaymentIntentActionParams{
    PaymentMethod: map[string]any{"type": "card", "card": map[string]any{ /* ... */ }},
})
refund, err := client.Refunds.Create(ctx, &airwallex.RefundCreateParams{
    PaymentIntentID: intent.ID, Amount: 5.00,
})
```

### Issue a card

```go
cardholder, err := client.IssuingCardholders.Create(ctx, &airwallex.CardholderCreateParams{
    Email: "employee@example.com",
    Individual: map[string]any{
        "name": map[string]any{"first_name": "Ada", "last_name": "Lovelace"},
    },
    Type: "INDIVIDUAL",
})
card, err := client.IssuingCards.Create(ctx, &airwallex.CardCreateParams{
    CardholderID: cardholder.CardholderID,
    FormFactor:   "VIRTUAL",
    CreatedBy:    "Ada Lovelace",
    Program:      map[string]any{"purpose": "COMMERCIAL"},
})
```

### Test flows in the sandbox

```go
// Demo environment only
client.Simulation.CreateDeposit(ctx, &airwallex.SimulationDepositParams{Amount: 1000, Currency: "USD"})
client.Simulation.TransitionTransfer(ctx, "tra_123", &airwallex.SimulationTransitionParams{NextStatus: "PAID"})
```

### Auto-pagination

```go
// Iterates page by page under the hood (Go 1.23 range-over-func)
for beneficiary, err := range client.Beneficiaries.All(ctx, nil) {
    if err != nil {
        return err
    }
    fmt.Println(beneficiary.EffectiveID(), beneficiary.Nickname)
}

// Or drive pages manually
page, err := client.Beneficiaries.List(ctx, &airwallex.BeneficiaryListParams{
    ListParams: airwallex.ListParams{PageSize: 100},
})
for page != nil {
    for _, b := range page.Items { /* ... */ }
    if !page.HasMore {
        break
    }
    page, err = page.Next(ctx)
}
```

### Webhooks

Verify and parse incoming notifications (get the secret when you create the webhook endpoint):

```go
import "github.com/Cyvid7-Darus10/airwallex-go/webhooks"

func handle(w http.ResponseWriter, r *http.Request) {
    payload, _ := io.ReadAll(r.Body) // raw bytes — do not re-serialise
    event, err := webhooks.ConstructEvent(
        payload,
        r.Header.Get("x-timestamp"),
        r.Header.Get("x-signature"),
        webhookSecret,
    )
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    if event.Name == "transfer.settled" { /* ... */ }
}
```

### Error handling

```go
transfer, err := client.Transfers.Retrieve(ctx, "tra_missing")
if err != nil {
    var apiErr *airwallex.Error
    var connErr *airwallex.ConnectionError
    switch {
    case errors.As(err, &apiErr):
        // HTTP status, Airwallex code/source, and x-request-id for support
        fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.RequestID)
    case errors.As(err, &connErr):
        // network failure / timeout — already retried automatically
    }
}
```

Rate limits (429) and transient 5xx are retried automatically before an error ever reaches you.

### Response metadata

Every resource and every page records the HTTP response it came from — quote `RequestID` when contacting Airwallex support:

```go
transfer, _ := client.Transfers.Retrieve(ctx, "tra_1")
fmt.Println(transfer.LastResponse.StatusCode, transfer.LastResponse.RequestID)
```

### Logging

Pass any `*slog.Logger` to see request outcomes, retries, and token refreshes at debug level. Only method, path, status, attempt, delay, and request id are logged — never credentials, tokens, headers, or bodies:

```go
client, err := airwallex.New(airwallex.WithLogger(slog.Default()))
```

### Calling endpoints the SDK doesn't wrap yet

Every list-params struct accepts extra query params via `ListParams.ExtraQuery`, every body-params struct accepts extra fields via `Params.ExtraParams`, and the client exposes a raw escape hatch with auth, retries, and error mapping intact:

```go
var disputes json.RawMessage
err := client.Request(ctx, "GET", "/api/v1/pa/payment_disputes",
    url.Values{"status": {"OPEN"}}, nil, &disputes)

// Per-call headers (e.g. a one-off x-api-version); Authorization stays SDK-managed
err = client.RequestWithHeaders(ctx, "GET", "/api/v1/pa/payment_disputes",
    nil, http.Header{"x-api-version": {"2020-01-01"}}, nil, &disputes)
```

Note on zero values: params structs use `omitempty`, so a `0` amount or `false` flag is omitted from the request. In the rare case you must send an explicit zero, put it in `ExtraParams`/`ExtraQuery`.

### Forward-compatible responses

Every response type embeds `airwallex.APIResource`, whose `Raw` field holds the exact JSON the API returned — so a field this SDK has no typed accessor for yet is still available:

```go
transfer, _ := client.Transfers.Retrieve(ctx, "tra_1")
var full map[string]any
json.Unmarshal(transfer.Raw, &full) // nothing is ever dropped
```

### Bring your own http.Client

```go
client, err := airwallex.New(
    airwallex.WithHTTPClient(&http.Client{
        Transport: proxyTransport, // proxies, custom TLS, tracing, ...
        Timeout:   30 * time.Second,
    }),
)
```

The SDK applies the base URL and default headers per request; it never mutates or closes a client you own.

### Connected accounts (platforms)

```go
client, err := airwallex.New(airwallex.WithOnBehalfOf("acct_connected_account_id")) // sets x-on-behalf-of
```

### Pinning an API version

```go
client, err := airwallex.New(airwallex.WithAPIVersion("2024-08-07")) // sets x-api-version on every request
```

## Examples

Runnable programs live in [`examples/`](examples/) — payouts, FX, and a webhook-verification server:

```bash
AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/payout
```

## Resources covered

| Resource | Methods |
|---|---|
| `client.Balances` | `Current`, `History`, `AllHistory` |
| `client.Transfers` | `Create`, `Retrieve`, `List`, `All`, `Cancel`, `Validate`, `ConfirmFunding` |
| `client.BatchTransfers` | `Create`, `Retrieve`, `List`, `All`, `AddItems`, `DeleteItems`, `Items`, `AllItems`, `Quote`, `Submit`, `Delete` |
| `client.WalletTransfers` | `Create`, `Retrieve`, `List`, `All` |
| `client.Payers` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `All`, `Validate` |
| `client.Beneficiaries` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `All`, `Validate` |
| `client.Conversions` | `Create`, `Retrieve`, `List`, `All` |
| `client.Rates` | `Current` |
| `client.FxQuotes` | `Create`, `Retrieve` |
| `client.ConversionAmendments` | `Create`, `Quote`, `Retrieve`, `List`, `All` |
| `client.PaymentIntents` | `Create`, `Retrieve`, `List`, `All`, `Confirm`, `ConfirmContinue`, `Capture`, `Cancel` |
| `client.Customers` | `Create`, `Retrieve`, `Update`, `List`, `All`, `GenerateClientSecret` |
| `client.Refunds` | `Create`, `Retrieve`, `List`, `All` |
| `client.IssuingCardholders` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `All` |
| `client.IssuingCards` | `Create`, `Retrieve`, `Update`, `Activate`, `Limits`, `List`, `All` |
| `client.IssuingTransactions` | `Retrieve`, `List`, `All` |
| `client.IssuingAuthorizations` | `Retrieve`, `List`, `All` |
| `client.Accounts` | `Retrieve` |
| `client.FinancialTransactions` | `Retrieve`, `List`, `All` |
| `client.Settlements` | `Retrieve`, `List`, `All` |
| `client.Simulation` | demo-only: deposit create/settle/reject/reverse, transfer/payment transitions |
| `client.GlobalAccounts` | `Create`, `Retrieve`, `Update`, `Close`, `List`, `All`, `Transactions`, `AllTransactions` |
| `client.Deposits` | `List`, `All` |
| `client.Reference` | `SupportedCurrencies`, `SettlementAccounts`, `InvalidConversionDates` |
| `client.WebhookEndpoints` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `All` |
| `webhooks` package | `VerifySignature`, `ConstructEvent` (+ `WithTolerance` variants) |

Coverage matches the [airwallex-python](https://github.com/Cyvid7-Darus10/airwallex-python) SDK v0.2.0 — contributions welcome for the remaining areas (disputes, payment consents, linked accounts, scale/platform APIs).

## Status

This SDK is **beta** software:

- The wrapped endpoints are grounded in Airwallex's published API spec and covered by tests (>90% coverage, race-detector clean), but they have not yet been exercised against every account configuration.
- Semantic versioning applies: breaking changes only in minor versions while `0.x`, and patch releases never change behavior.
- Response types tolerate unknown fields and preserve the raw JSON, so new Airwallex API versions won't break parsing.
- Test in the `Demo` environment before pointing at production, and pin the version in your `go.mod`.

## Development

```bash
make check    # gofmt + vet + golangci-lint + race tests (what CI runs)
make cover    # coverage report
```

## Disclaimer

This project is an independent, unofficial SDK maintained by the community. It is not affiliated with, endorsed by, sponsored by, or supported by Airwallex Pty Ltd. "Airwallex" and related marks are trademarks of Airwallex Pty Ltd; they are used here solely to indicate API compatibility. This software is provided "as is" under the MIT license — review the [SECURITY policy](SECURITY.md) and test against the demo environment before moving real money. If you need vendor support or SLAs, use the [official Node.js SDK](https://www.npmjs.com/package/@airwallex/node-sdk).

## License

[MIT](LICENSE)
