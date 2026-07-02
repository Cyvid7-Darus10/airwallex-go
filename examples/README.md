# Examples

Runnable programs demonstrating the SDK against the Airwallex demo
environment. Each reads credentials from `AIRWALLEX_CLIENT_ID` /
`AIRWALLEX_API_KEY`.

| Example | What it shows |
|---|---|
| [`payout`](payout/) | Balances, beneficiaries, validate + create a transfer |
| [`fx`](fx/) | Indicative rate, lockable quote, conversion |
| [`webhook-server`](webhook-server/) | Verifying webhook signatures in an HTTP handler |

```bash
AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/payout
```

Use a **demo** account — these move (sandbox) money.
