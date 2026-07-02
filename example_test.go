package airwallex_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	airwallex "github.com/Cyvid7-Darus10/airwallex-go"
)

// Construct a client for the demo (sandbox) environment. Credentials can
// also come from the AIRWALLEX_CLIENT_ID / AIRWALLEX_API_KEY environment
// variables, in which case New() needs no credential options at all.
func ExampleNew() {
	client, err := airwallex.New(
		airwallex.WithClientID("your_client_id"),
		airwallex.WithAPIKey("your_api_key"),
		airwallex.WithEnv(airwallex.Demo),
	)
	if err != nil {
		log.Fatal(err)
	}
	balances, err := client.Balances.Current(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, balance := range balances {
		fmt.Println(balance.Currency, balance.AvailableAmount)
	}
}

// Send a payout. RequestID is auto-generated when empty, so the call is
// idempotent even across the SDK's automatic retries.
func ExampleTransfersService_Create() {
	client, _ := airwallex.New(airwallex.WithEnv(airwallex.Demo))
	transfer, err := client.Transfers.Create(context.Background(), &airwallex.TransferCreateParams{
		BeneficiaryID:    "ben_abc123",
		SourceCurrency:   "USD",
		TransferCurrency: "PHP",
		TransferAmount:   5000,
		TransferMethod:   "LOCAL",
		Reference:        "Invoice 42",
		Reason:           "professional_service_fees",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(transfer.ID, transfer.Status)
}

// Walk every beneficiary across every page with one loop.
func ExampleBeneficiariesService_All() {
	client, _ := airwallex.New(airwallex.WithEnv(airwallex.Demo))
	for beneficiary, err := range client.Beneficiaries.All(context.Background(), nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(beneficiary.BeneficiaryID, beneficiary.Nickname)
	}
}

// Inspect a typed API error, including the request id to quote to
// Airwallex support.
func ExampleError() {
	client, _ := airwallex.New(airwallex.WithEnv(airwallex.Demo))
	_, err := client.Transfers.Retrieve(context.Background(), "tra_missing")
	var apiErr *airwallex.Error
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.RequestID)
	}
}

// Call an endpoint the SDK has no typed wrapper for, with auth, retries,
// and error mapping intact.
func ExampleClient_Request() {
	client, _ := airwallex.New(airwallex.WithEnv(airwallex.Demo))
	var out map[string]any
	err := client.Request(context.Background(), "GET",
		"/api/v1/reference/supported_currencies", nil, nil, &out)
	if err != nil {
		log.Fatal(err)
	}
}
