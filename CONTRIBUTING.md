# Contributing

Thanks for helping improve the unofficial Airwallex Go SDK!

## Setup

```bash
git clone https://github.com/Cyvid7-Darus10/airwallex-go
cd airwallex-go
go test ./...
```

No dependencies to install — the SDK is standard library only, and stays that way.

## Before opening a PR

```bash
gofmt -l .              # formatting (must print nothing)
go vet ./...            # static analysis
golangci-lint run       # lint
go test -race -cover ./...  # tests (coverage must stay >= 85%)
```

All four must pass; CI runs them on the last three Go releases.

## Guidelines

- **Tests first.** Every behavior change needs a test that fails without it.
  Tests use `httptest.Server` — no network calls.
- **Ground new endpoints in the official spec** (`https://www.airwallex.com/docs/api/schema.json`)
  rather than guessing field names, and keep behavior in parity with
  [airwallex-python](https://github.com/Cyvid7-Darus10/airwallex-python).
- Response types embed `APIResource` and are additive-only — type the
  documented fields, let unknown fields live in `.Raw`.
- Money-moving `Create` calls must go through `idempotentBody` so a
  `request_id` is always present.
- Path parameters must go through `pathEscape`.
- Conventional commits: `feat: ...`, `fix: ...`, `docs: ...`, etc.

## Releases (maintainers)

1. Bump `Version` in `transport.go`, update `CHANGELOG.md`.
2. Tag: `git tag v<version> && git push --tags` — Go modules publish via tags.
3. Verify the new version appears on [pkg.go.dev](https://pkg.go.dev/github.com/Cyvid7-Darus10/airwallex-go).
