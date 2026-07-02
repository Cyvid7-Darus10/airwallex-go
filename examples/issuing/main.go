// Command issuing demonstrates issuing a virtual card in the Airwallex
// demo environment: create a cardholder, issue a card, inspect its
// limits, and list its transactions. Requires Issuing to be enabled.
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/issuing
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

	// Unique per run — cardholder emails must not collide.
	suffix := time.Now().Unix()
	cardholder, err := client.IssuingCardholders.Create(ctx, &airwallex.CardholderCreateParams{
		Email:        fmt.Sprintf("ada+%d@example.com", suffix),
		MobileNumber: fmt.Sprintf("+65%08d", suffix%100000000),
		Type:         "INDIVIDUAL",
		Individual: map[string]any{
			"name": map[string]any{
				"first_name": "Ada",
				"last_name":  "Lovelace",
			},
			"date_of_birth":            "1990-12-10",
			"express_consent_obtained": "yes",
			"address": map[string]any{
				"city":     "Singapore",
				"country":  "SG",
				"line1":    "1 Raffles Place",
				"postcode": "048616",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("cardholder: %w", err)
	}
	fmt.Printf("cardholder %s — status %s\n", cardholder.CardholderID, cardholder.Status)

	// RequestID is auto-generated: a retry never issues two cards.
	card, err := client.IssuingCards.Create(ctx, &airwallex.CardCreateParams{
		CardholderID: cardholder.CardholderID,
		FormFactor:   "VIRTUAL",
		Params: airwallex.Params{ExtraParams: map[string]any{
			"is_personalized": false,
		}},
		IssueTo:   "INDIVIDUAL",
		CreatedBy: "Ada Lovelace",
		Program:   map[string]any{"purpose": "COMMERCIAL"},
		AuthorizationControls: map[string]any{
			"allowed_transaction_count": "MULTIPLE",
			"transaction_limits": map[string]any{
				"currency": "USD",
				"limits": []map[string]any{
					{"amount": 1000, "interval": "PER_TRANSACTION"},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("card: %w", err)
	}
	fmt.Printf("card %s — status %s\n", card.CardID, card.CardStatus)

	limits, err := client.IssuingCards.Limits(ctx, card.CardID)
	if err != nil {
		return fmt.Errorf("limits: %w", err)
	}
	fmt.Printf("card limits: %s %v\n", limits.Currency, limits.Limits)

	// Card transactions arrive as the card is used; iterate them all.
	count := 0
	for _, err := range client.IssuingTransactions.All(ctx, &airwallex.IssuingTransactionListParams{
		CardID: card.CardID,
	}) {
		if err != nil {
			return fmt.Errorf("transactions: %w", err)
		}
		count++
	}
	fmt.Printf("card has %d transactions so far\n", count)
	return nil
}
