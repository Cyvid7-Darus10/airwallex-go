// Command payment-acceptance demonstrates collecting a payment in the
// Airwallex demo environment: create a customer, create a payment intent,
// and refund it. Requires Payment Acceptance to be enabled on the account.
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/payment-acceptance
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

	// A customer lets you save payment details for repeat shoppers.
	customer, err := client.Customers.Create(ctx, &airwallex.CustomerCreateParams{
		MerchantCustomerID: fmt.Sprintf("example-%d", time.Now().Unix()),
		FirstName:          "Ada",
		LastName:           "Lovelace",
		Email:              "ada@example.com",
	})
	if err != nil {
		return fmt.Errorf("customer: %w", err)
	}
	fmt.Printf("customer %s created\n", customer.ID)

	// The intent is what your checkout confirms (browser SDK, mobile SDK,
	// or an API confirm with a payment method).
	intent, err := client.PaymentIntents.Create(ctx, &airwallex.PaymentIntentCreateParams{
		Amount:          25.00,
		Currency:        "USD",
		MerchantOrderID: fmt.Sprintf("order-%d", time.Now().Unix()),
		CustomerID:      customer.ID,
	})
	if err != nil {
		return fmt.Errorf("intent: %w", err)
	}
	fmt.Printf("payment intent %s — status %s, client_secret for your frontend: %.12s…\n",
		intent.ID, intent.Status, intent.ClientSecret)

	// Once an intent has succeeded (confirmed by your checkout), refund it
	// fully or partially. RequestID is auto-generated, so a retry never
	// refunds twice.
	refund, err := client.Refunds.Create(ctx, &airwallex.RefundCreateParams{
		PaymentIntentID: intent.ID,
		Amount:          5.00,
		Reason:          "requested_by_customer",
	})
	if err != nil {
		// Refunding an unconfirmed intent fails — expected when running
		// this example without a checkout step in between.
		fmt.Printf("refund (expected to fail before confirmation): %v\n", err)
		return nil
	}
	fmt.Printf("refund %s — status %s\n", refund.ID, refund.Status)
	return nil
}
