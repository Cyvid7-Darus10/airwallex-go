// Command patterns demonstrates the SDK's cross-cutting features: typed
// error handling, response metadata, auto-pagination, debug logging, and
// the escape hatch for endpoints without typed wrappers.
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/patterns
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	airwallex "github.com/Cyvid7-Darus10/airwallex-go"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// WithLogger emits request outcomes, retries, and token refreshes at
	// debug level — credentials are never logged.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	client, err := airwallex.New(
		airwallex.WithEnv(airwallex.Demo),
		airwallex.WithLogger(logger),
		airwallex.WithMaxRetries(3),
	)
	if err != nil {
		return err
	}

	// --- Typed errors ---------------------------------------------------
	_, err = client.Transfers.Retrieve(ctx, "tra_does_not_exist")
	var apiErr *airwallex.Error
	var connErr *airwallex.ConnectionError
	switch {
	case errors.As(err, &apiErr):
		// Quote RequestID to Airwallex support; Raw holds the full error
		// body (validation failures carry a per-field errors object).
		fmt.Printf("API error: status=%d code=%s request_id=%q\nbody: %s\n",
			apiErr.StatusCode, apiErr.Code, apiErr.RequestID, apiErr.Raw)
	case errors.As(err, &connErr):
		fmt.Printf("network failure (already retried): %v\n", connErr)
	}

	// --- Response metadata ----------------------------------------------
	balances, err := client.Balances.Current(ctx)
	if err != nil {
		return err
	}
	if len(balances) > 0 {
		meta := balances[0].LastResponse
		fmt.Printf("balances came from HTTP %d (request id %q)\n", meta.StatusCode, meta.RequestID)
	}

	// --- Pagination: iterator and manual --------------------------------
	total := 0
	for _, err := range client.Beneficiaries.All(ctx, nil) {
		if err != nil {
			return err
		}
		total++
	}
	fmt.Printf("iterator walked %d beneficiaries\n", total)

	page, err := client.Beneficiaries.List(ctx, &airwallex.BeneficiaryListParams{
		ListParams: airwallex.ListParams{PageSize: 10},
	})
	if err != nil {
		return err
	}
	fmt.Printf("manual paging: %d on the first page, more=%t\n", len(page.Items), page.HasMore)

	// --- Escape hatch ----------------------------------------------------
	// Call any endpoint — auth, retries, and error mapping still apply.
	var currencies json.RawMessage
	err = client.Request(ctx, http.MethodGet, "/api/v1/reference/supported_currencies",
		url.Values{}, nil, &currencies)
	if err != nil {
		return err
	}
	fmt.Printf("escape hatch fetched %d bytes of reference data\n", len(currencies))

	// Unknown body fields go in ExtraParams; unknown filters in ExtraQuery.
	_, err = client.Transfers.List(ctx, &airwallex.TransferListParams{
		ListParams: airwallex.ListParams{
			ExtraQuery: url.Values{"short_reference_id": {"REF123"}},
		},
	})
	if err != nil {
		return err
	}
	fmt.Println("extra query filter accepted")
	return nil
}
