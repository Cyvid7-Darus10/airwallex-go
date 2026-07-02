# Examples

Runnable programs demonstrating the SDK against the Airwallex demo
environment. Each reads credentials from `AIRWALLEX_CLIENT_ID` /
`AIRWALLEX_API_KEY`:

```bash
AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/payout
```

| Example | What it shows |
|---|---|
| [`payout`](payout/) | Balances → beneficiaries → validate → create a transfer |
| [`fx`](fx/) | Indicative rate, lockable quote, conversion |
| [`collect-funds`](collect-funds/) | Global accounts, simulated deposits, balance ledger |
| [`payment-acceptance`](payment-acceptance/) | Customer, payment intent, refund |
| [`issuing`](issuing/) | Cardholder, virtual card, limits, card transactions |
| [`webhook-server`](webhook-server/) | Verifying webhook signatures in an HTTP handler |
| [`patterns`](patterns/) | Typed errors, response metadata, pagination, logging, escape hatch |

Use a **demo** account — several examples move (sandbox) money. Payment
acceptance and issuing must be enabled on the account for those examples.
