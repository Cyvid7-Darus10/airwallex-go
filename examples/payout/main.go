// Command payout demonstrates the core payout flow against the Airwallex
// demo environment: check balances, list beneficiaries, validate, and
// create a transfer.
//
// Run with your demo credentials:
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/payout
package main

import (
	"context"
	"fmt"
	"log"
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

	client, err := airwallex.New(airwallex.WithEnv(airwallex.Demo))
	if err != nil {
		return err
	}

	balances, err := client.Balances.Current(ctx)
	if err != nil {
		return fmt.Errorf("balances: %w", err)
	}
	for _, balance := range balances {
		if balance.TotalAmount > 0 {
			fmt.Printf("balance: %s %.2f available\n", balance.Currency, balance.AvailableAmount)
		}
	}

	page, err := client.Beneficiaries.List(ctx, &airwallex.BeneficiaryListParams{
		ListParams: airwallex.ListParams{PageSize: 10},
	})
	if err != nil {
		return fmt.Errorf("beneficiaries: %w", err)
	}
	if len(page.Items) == 0 {
		return fmt.Errorf("no beneficiaries on this account; create one in the demo dashboard first")
	}
	beneficiary := page.Items[0]
	fmt.Printf("paying: %s (%s)\n", beneficiary.Nickname, beneficiary.EffectiveID())

	params := &airwallex.TransferCreateParams{
		BeneficiaryID:    beneficiary.EffectiveID(),
		SourceCurrency:   "GBP",
		TransferCurrency: "GBP",
		TransferAmount:   12.34,
		TransferMethod:   "LOCAL",
		Reference:        "Example payout",
		Reason:           "professional_service_fees",
	}

	// Dry-run first; nothing is executed.
	if _, err := client.Transfers.Validate(ctx, params); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	// RequestID is auto-generated, so this create is idempotent even
	// across the SDK's automatic retries.
	transfer, err := client.Transfers.Create(ctx, params)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	fmt.Printf("transfer %s created — status %s (request_id %s)\n",
		transfer.ID, transfer.Status, transfer.RequestID)
	return nil
}
