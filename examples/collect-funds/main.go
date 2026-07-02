// Command collect-funds demonstrates the receivables flow in the
// Airwallex demo environment: global accounts, simulated incoming
// deposits, and the resulting balance and ledger entries.
//
//	AIRWALLEX_CLIENT_ID=... AIRWALLEX_API_KEY=... go run ./examples/collect-funds
package main

import (
	"context"
	"encoding/json"
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

	// Global accounts are the local bank details your payers send funds to.
	accounts, err := client.GlobalAccounts.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("global accounts: %w", err)
	}
	if len(accounts.Items) == 0 {
		return fmt.Errorf("no global accounts; open one in the demo dashboard first")
	}
	account := accounts.Items[0]
	bank := account.InstitutionName
	if bank == "" && account.Institution != nil {
		bank = account.Institution.Name
	}
	currency := account.PrimaryCurrency()
	fmt.Printf("global account %s — %s account %q at %s\n",
		account.ID, currency, account.AccountNumber, bank)

	// Simulate a payer sending money in (demo environment only). Deposits
	// auto-settle in the sandbox.
	raw, err := client.Simulation.CreateDeposit(ctx, &airwallex.SimulationDepositParams{
		Amount:          150,
		Currency:        currency,
		GlobalAccountID: account.ID,
		Reference:       "Invoice 42",
	})
	if err != nil {
		return fmt.Errorf("simulated deposit: %w", err)
	}
	var deposit struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(raw, &deposit)
	fmt.Printf("simulated deposit %s — status %s\n", deposit.ID, deposit.Status)

	// The deposit appears in the deposits list and the balance ledger.
	deposits, err := client.Deposits.List(ctx, &airwallex.DepositListParams{
		ListParams: airwallex.ListParams{PageSize: 10},
	})
	if err != nil {
		return fmt.Errorf("deposits: %w", err)
	}
	for _, d := range deposits.Items {
		fmt.Printf("deposit: %s %.2f — %s (settled %s)\n", d.Currency, d.Amount, d.Status, d.SettledAt)
	}

	history, err := client.Balances.History(ctx, &airwallex.BalanceHistoryParams{
		Currency: currency,
	})
	if err != nil {
		return fmt.Errorf("balance history: %w", err)
	}
	for _, row := range history.Items {
		fmt.Printf("ledger: %s %+.2f → balance %.2f (%s)\n",
			row.Currency, row.Amount, row.Balance, row.SourceType)
	}
	return nil
}
