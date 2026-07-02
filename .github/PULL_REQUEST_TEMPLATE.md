## What

<!-- One sentence: what does this PR change? -->

## Why

<!-- The motivation — the diff shows the what. -->

## Checklist

- [ ] `make check` passes (gofmt, vet, golangci-lint, race tests)
- [ ] New behavior has a test that fails without the change
- [ ] Endpoint changes are grounded in the [API spec](https://www.airwallex.com/docs/api/schema.json), not guessed
- [ ] Money-moving creates go through `idempotentBody`; path parameters through `pathEscape`
- [ ] CHANGELOG.md updated under **Unreleased**
