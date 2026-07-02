// Command fx demonstrates the FX flow against the Airwallex demo
// environment: indicative rate, lockable quote, and a conversion.
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/fx
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

	// Indicative rate — no funds move.
	rate, err := client.Rates.Current(ctx, &airwallex.RateCurrentParams{
		BuyCurrency: "USD", SellCurrency: "SGD", BuyAmount: 1000,
	})
	if err != nil {
		return fmt.Errorf("rate: %w", err)
	}
	fmt.Printf("indicative %s: %v\n", rate.CurrencyPair, rate.Rate)

	// Lock the rate for an hour.
	quote, err := client.FxQuotes.Create(ctx, &airwallex.FxQuoteCreateParams{
		BuyCurrency: "USD", SellCurrency: "SGD", SellAmount: 100, Validity: "HR_1",
	})
	if err != nil {
		return fmt.Errorf("quote: %w", err)
	}
	fmt.Printf("locked quote %s at %v until %s\n", quote.QuoteID, quote.ClientRate, quote.ExpiresAt)

	// Execute a conversion (demo funds).
	conversion, err := client.Conversions.Create(ctx, &airwallex.ConversionCreateParams{
		BuyCurrency:   "USD",
		SellCurrency:  "SGD",
		SellAmount:    100,
		TermAgreement: true,
		Reason:        "example conversion",
	})
	if err != nil {
		return fmt.Errorf("conversion: %w", err)
	}
	fmt.Printf("conversion %s — status %s, %v %s → %v %s\n",
		conversion.ConversionID, conversion.Status,
		conversion.SellAmount, conversion.SellCurrency,
		conversion.BuyAmount, conversion.BuyCurrency)
	return nil
}
