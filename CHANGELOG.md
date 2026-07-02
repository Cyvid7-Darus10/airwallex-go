# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the project is `0.x`, breaking changes are only made in minor versions and
patch releases never change behavior.

## [Unreleased]

## [0.2.4] - 2026-07-02

### Added

- `GlobalAccount` now types the structure current API versions return:
  `Institution`, `RequiredFeatures` / `SupportedFeatures` (with routing
  codes), `AccountType`, plus a `PrimaryCurrency()` helper spanning old
  and new shapes. Legacy flat fields remain.
- Four more runnable examples — `collect-funds`, `payment-acceptance`,
  `issuing`, and `patterns` (typed errors, metadata, pagination, logging,
  escape hatch). All seven examples are verified against the live demo
  API, which also live-validated the issuing and payment-acceptance
  create flows.

## [0.2.3] - 2026-07-02

### Added

- Package documentation (`doc.go`), runnable examples under `examples/`
  (payout, FX, webhook server), `Makefile`, PR template, and CODEOWNERS.
  No behavior changes.

## [0.2.2] - 2026-07-02

### Added

- `Error.Raw` exposes the full error response body, so validation
  failures' per-field `errors` object is available beyond `Message`.
- `Beneficiary.ID` / `Beneficiary.TransferMethods` (current API versions
  return these instead of `beneficiary_id` / `payment_methods`) and a
  `Beneficiary.EffectiveID()` helper covering both.
- `Deposit.Type` and `Deposit.SettledAt`.

### Fixed

- `Transfers.Validate` now auto-generates a `request_id` — current API
  versions require one even to validate (nothing is executed either way).

All verified against the live demo API, including a settled end-to-end
conversion (FX quote → conversion → SETTLED) and the deposit lifecycle.

## [0.2.1] - 2026-07-02

### Fixed

- `RateQuote` now types the fields current API versions actually return
  (`rate`, `buy_currency`, `buy_amount`, `sell_currency`, `sell_amount`,
  `conversion_date`, `created_at`), verified against the live demo API.
  The legacy `client_rate`/`mid_rate` fields remain for older API versions.
- `FxQuote` now types `quote_id` (current API versions) alongside the
  legacy `id`.

## [0.2.0] - 2026-07-02

### Added

- `LastResponse` metadata (status code, `x-request-id`, headers) on every
  resource, every list item, and every `Page` — parity with stripe-go's
  `LastResponse`.
- `WithLogger(*slog.Logger)` option: opt-in debug logging of request
  outcomes, retries, and token refreshes. Credentials, tokens, headers, and
  bodies are never logged.
- `Client.RequestWithHeaders`: the escape hatch can now send per-call
  headers (parity with the Python SDK's `client.request(headers=...)`);
  the `Authorization` header always remains SDK-managed.
- Runnable examples on pkg.go.dev for the client, payouts, auto-pagination,
  error handling, the escape hatch, and webhook verification.
- CI: `govulncheck` job, and the test matrix now includes Go 1.23 (the
  declared minimum) alongside the last three releases.
- Lint: revive `exported`/`package-comments` rules enforce doc comments on
  the entire public API.

## [0.1.0] - 2026-07-02

### Added

- Initial release, matching the capabilities of [airwallex-python](https://github.com/Cyvid7-Darus10/airwallex-python) v0.2.0.
- `airwallex.New` client with functional options (`WithClientID`, `WithAPIKey`,
  `WithEnv`, `WithBaseURL`, `WithAPIVersion`, `WithOnBehalfOf`, `WithTimeout`,
  `WithMaxRetries`, `WithHTTPClient`); credentials default to the
  `AIRWALLEX_CLIENT_ID` / `AIRWALLEX_API_KEY` environment variables.
- Automatic bearer-token management: login on first use, thread-safe cache,
  refresh 60s before expiry, single re-login on 401.
- Automatic retries with full-jitter exponential backoff on 408/429/5xx and
  network failures, honouring `Retry-After` (delta-seconds and HTTP-date).
  409 business conflicts are never retried. The login endpoint shares the
  same retry budget.
- Idempotency: `request_id` auto-generated (UUIDv4) for money-moving create
  calls and reused byte-for-byte across retries.
- 25 services: Balances, Transfers, BatchTransfers, WalletTransfers,
  Beneficiaries, Payers, Conversions, Rates, FxQuotes, ConversionAmendments,
  GlobalAccounts, Deposits, PaymentIntents, Customers, Refunds,
  IssuingCardholders, IssuingCards, IssuingTransactions,
  IssuingAuthorizations, Accounts, FinancialTransactions, Settlements,
  Reference, WebhookEndpoints, Simulation (demo-only).
- Auto-pagination: `List` returns a `Page[T]`; `All` returns a Go 1.23
  range-over-func iterator; `Page.Next` for manual paging. Defensive
  termination when `has_more` is true but a page is empty.
- `webhooks` package: `ConstructEvent` / `VerifySignature` with constant-time
  HMAC-SHA256 comparison, replay tolerance (default 5 minutes), and support
  for second- and millisecond-precision timestamps.
- Typed errors: `*airwallex.Error` (status, code, source, request id) and
  `*airwallex.ConnectionError` for transport failures.
- Forward-compatible responses: every resource embeds `APIResource` and keeps
  the raw response JSON in `.Raw`.
- Escape hatch `client.Request` for endpoints without typed wrappers, plus
  `Params.ExtraParams` / `ListParams.ExtraQuery` on every params struct.
- Security hardening carried over from the Python SDK's review: URL path
  escaping of resource ids, HTTPS-only base URLs (plain http restricted to
  loopback), credential redaction in `String()`/`GoString()`, typed errors
  for non-JSON responses, custom `http.Client` support without mutation.

[Unreleased]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.2.4...HEAD
[0.2.4]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Cyvid7-Darus10/airwallex-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Cyvid7-Darus10/airwallex-go/releases/tag/v0.1.0
