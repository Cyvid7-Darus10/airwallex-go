// Package airwallex is an unofficial Go SDK for the Airwallex API —
// payouts, FX, balances, global accounts, payment acceptance, issuing,
// and webhooks.
//
// # Getting started
//
// Create a Client with New; credentials default to the
// AIRWALLEX_CLIENT_ID / AIRWALLEX_API_KEY environment variables:
//
//	client, err := airwallex.New(
//	    airwallex.WithClientID("..."),
//	    airwallex.WithAPIKey("..."),
//	    airwallex.WithEnv(airwallex.Demo), // Production is the default
//	)
//	balances, err := client.Balances.Current(ctx)
//
// Authentication happens lazily on the first request; the bearer token is
// cached and refreshed automatically before it expires.
//
// # Reliability
//
// Transient failures (408/429/5xx and network errors) are retried with
// full-jitter exponential backoff, honouring Retry-After in both
// delta-seconds and HTTP-date form. 409 business conflicts are never
// retried. Money-moving creates carry an auto-generated request_id that is
// re-sent byte-for-byte on every retry, so Airwallex never executes the
// same operation twice.
//
// # Responses
//
// Every response type embeds APIResource: the exact JSON the API returned
// is preserved in Raw (fields from newer API versions are never lost), and
// LastResponse records the HTTP status, x-request-id, and headers.
//
// # Errors
//
// API failures are returned as *Error (status, Airwallex code, source,
// request id, and the raw error body); transport failures as
// *ConnectionError. Both work with errors.As.
//
// # Pagination
//
// List methods return a Page; All methods return a Go 1.23 iterator that
// fetches pages lazily:
//
//	for b, err := range client.Beneficiaries.All(ctx, nil) {
//	    if err != nil { return err }
//	    fmt.Println(b.EffectiveID())
//	}
//
// # Webhooks
//
// The webhooks subpackage verifies webhook signatures with constant-time
// comparison and replay protection; see
// github.com/Cyvid7-Darus10/airwallex-go/webhooks.
//
// # Disclaimer
//
// This library is not affiliated with, endorsed by, or supported by
// Airwallex Pty Ltd. "Airwallex" is their trademark, used here only to
// describe compatibility.
package airwallex
