# Security Policy

## Reporting a vulnerability

Please do **not** open a public issue for security problems. Email
**cyrus@pastelero.ph** with the details, or use GitHub's
[private vulnerability reporting](https://github.com/Cyvid7-Darus10/airwallex-go/security/advisories/new).
You should receive a response within 72 hours.

## Supported versions

Only the latest released version receives security fixes.

## Design notes for integrators

- API credentials are read from functional options or the
  `AIRWALLEX_CLIENT_ID` / `AIRWALLEX_API_KEY` environment variables — never
  hardcode them.
- The SDK redacts credentials and bearer tokens from `String()`, `GoString()`
  (`%v`/`%+v`/`%#v`), and every error it returns. Avoid logging raw
  `*http.Request` objects you obtain by other means.
- The base URL must be HTTPS (plain HTTP is allowed only for localhost mocks).
- Webhook handlers must verify signatures with `webhooks.ConstructEvent`
  using the raw request body; verification uses a constant-time comparison
  and replay protection rejects stale timestamps.
- Resource ids are percent-encoded before URL interpolation, so untrusted
  ids cannot re-route a request to a different endpoint.
