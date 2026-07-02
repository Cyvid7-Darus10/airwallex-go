---
name: Feature request
about: Request coverage of an Airwallex endpoint or an SDK improvement
labels: enhancement
---

**What do you need?**
e.g. "wrap POST /api/v1/payment_intents/create" or "expose response headers".

**Workaround today**
Note: `client.Request(ctx, "METHOD", "/api/v1/...", params, body, &out)` already
reaches any endpoint with auth, retries, and error mapping — tell us what the
raw escape hatch doesn't cover for you.

**Link to Airwallex docs**
https://www.airwallex.com/docs/api
