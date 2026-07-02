package airwallex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recorder captures the last non-login request the server saw.
type recorder struct {
	method   string
	path     string // escaped path, exactly as sent
	query    string
	body     map[string]any
	rawBody  []byte
	response string
}

func newRecordingServer(t *testing.T) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == loginPath {
			loginOK(w)
			return
		}
		rec.method = r.Method
		rec.path = r.URL.EscapedPath()
		rec.query = r.URL.RawQuery
		rec.rawBody, _ = io.ReadAll(r.Body)
		rec.body = nil
		if len(rec.rawBody) > 0 {
			_ = json.Unmarshal(rec.rawBody, &rec.body)
		}
		response := rec.response
		if response == "" {
			response = `{"has_more":false,"items":[]}`
		}
		fmt.Fprint(w, response)
	}))
	t.Cleanup(server.Close)
	return server, rec
}

// evilID exercises path-parameter escaping: unescaped it would traverse to
// a sibling endpoint.
const evilID = "../create"

var escapedEvilID = "..%2Fcreate"

type routingTest struct {
	name       string
	response   string
	call       func() error
	wantMethod string
	wantPath   string
	wantReqID  bool // auto-generated request_id must be present
}

func buildRoutingTests(ctx context.Context, client *Client) []routingTest {
	lp := ListParams{PageSize: 10}
	return []routingTest{
		// Accounts & Balances
		{"Accounts.Retrieve", "", func() error { _, err := client.Accounts.Retrieve(ctx); return err },
			"GET", "/api/v1/account", false},
		{"Balances.Current", "[]", func() error { _, err := client.Balances.Current(ctx); return err },
			"GET", "/api/v1/balances/current", false},
		{"Balances.History", "", func() error { _, err := client.Balances.History(ctx, nil); return err },
			"GET", "/api/v1/balances/history", false},

		// Transfers
		{"Transfers.Create", "", func() error { _, err := client.Transfers.Create(ctx, &TransferCreateParams{}); return err },
			"POST", "/api/v1/transfers/create", true},
		{"Transfers.Retrieve", "", func() error { _, err := client.Transfers.Retrieve(ctx, evilID); return err },
			"GET", "/api/v1/transfers/" + escapedEvilID, false},
		{"Transfers.List", "", func() error { _, err := client.Transfers.List(ctx, nil); return err },
			"GET", "/api/v1/transfers", false},
		{"Transfers.Cancel", "", func() error { _, err := client.Transfers.Cancel(ctx, evilID); return err },
			"POST", "/api/v1/transfers/" + escapedEvilID + "/cancel", false},
		{"Transfers.Validate", "", func() error { _, err := client.Transfers.Validate(ctx, &TransferCreateParams{}); return err },
			"POST", "/api/v1/transfers/validate", true},
		{"Transfers.ConfirmFunding", "", func() error {
			_, err := client.Transfers.ConfirmFunding(ctx, "tra_1", &TransferConfirmFundingParams{FundingSourceID: "fs_1"})
			return err
		}, "POST", "/api/v1/transfers/tra_1/confirm_funding", false},

		// Batch transfers
		{"BatchTransfers.Create", "", func() error {
			_, err := client.BatchTransfers.Create(ctx, &BatchTransferCreateParams{})
			return err
		}, "POST", "/api/v1/batch_transfers/create", true},
		{"BatchTransfers.Retrieve", "", func() error { _, err := client.BatchTransfers.Retrieve(ctx, evilID); return err },
			"GET", "/api/v1/batch_transfers/" + escapedEvilID, false},
		{"BatchTransfers.List", "", func() error { _, err := client.BatchTransfers.List(ctx, nil); return err },
			"GET", "/api/v1/batch_transfers", false},
		{"BatchTransfers.AddItems", "", func() error {
			_, err := client.BatchTransfers.AddItems(ctx, "bat_1", []map[string]any{{"beneficiary_id": "ben_1"}})
			return err
		}, "POST", "/api/v1/batch_transfers/bat_1/add_items", false},
		{"BatchTransfers.DeleteItems", "", func() error {
			_, err := client.BatchTransfers.DeleteItems(ctx, "bat_1", []string{"item_1"})
			return err
		}, "POST", "/api/v1/batch_transfers/bat_1/delete_items", false},
		{"BatchTransfers.Items", "", func() error { _, err := client.BatchTransfers.Items(ctx, "bat_1", &lp); return err },
			"GET", "/api/v1/batch_transfers/bat_1/items", false},
		{"BatchTransfers.Quote", "", func() error { _, err := client.BatchTransfers.Quote(ctx, "bat_1", nil); return err },
			"POST", "/api/v1/batch_transfers/bat_1/quote", false},
		{"BatchTransfers.Submit", "", func() error { _, err := client.BatchTransfers.Submit(ctx, "bat_1"); return err },
			"POST", "/api/v1/batch_transfers/bat_1/submit", false},
		{"BatchTransfers.Delete", "", func() error { _, err := client.BatchTransfers.Delete(ctx, "bat_1"); return err },
			"POST", "/api/v1/batch_transfers/bat_1/delete", false},

		// Wallet transfers
		{"WalletTransfers.Create", "", func() error {
			_, err := client.WalletTransfers.Create(ctx, &WalletTransferCreateParams{})
			return err
		}, "POST", "/api/v1/wallet_transfers/create", true},
		{"WalletTransfers.Retrieve", "", func() error { _, err := client.WalletTransfers.Retrieve(ctx, "wt_1"); return err },
			"GET", "/api/v1/wallet_transfers/wt_1", false},
		{"WalletTransfers.List", "", func() error { _, err := client.WalletTransfers.List(ctx, nil); return err },
			"GET", "/api/v1/wallet_transfers", false},

		// Beneficiaries (no request_id on this endpoint)
		{"Beneficiaries.Create", "", func() error {
			_, err := client.Beneficiaries.Create(ctx, &BeneficiaryCreateParams{Nickname: "n"})
			return err
		}, "POST", "/api/v1/beneficiaries/create", false},
		{"Beneficiaries.Retrieve", "", func() error { _, err := client.Beneficiaries.Retrieve(ctx, evilID); return err },
			"GET", "/api/v1/beneficiaries/" + escapedEvilID, false},
		{"Beneficiaries.Update", "", func() error {
			_, err := client.Beneficiaries.Update(ctx, "ben_1", &BeneficiaryCreateParams{})
			return err
		}, "POST", "/api/v1/beneficiaries/update/ben_1", false},
		{"Beneficiaries.Delete", "", func() error { return client.Beneficiaries.Delete(ctx, "ben_1") },
			"POST", "/api/v1/beneficiaries/delete/ben_1", false},
		{"Beneficiaries.List", "", func() error { _, err := client.Beneficiaries.List(ctx, nil); return err },
			"GET", "/api/v1/beneficiaries", false},
		{"Beneficiaries.Validate", "", func() error {
			_, err := client.Beneficiaries.Validate(ctx, &BeneficiaryCreateParams{})
			return err
		}, "POST", "/api/v1/beneficiaries/validate", false},

		// Payers (no request_id on this endpoint)
		{"Payers.Create", "", func() error { _, err := client.Payers.Create(ctx, &PayerCreateParams{}); return err },
			"POST", "/api/v1/payers/create", false},
		{"Payers.Retrieve", "", func() error { _, err := client.Payers.Retrieve(ctx, "pay_1"); return err },
			"GET", "/api/v1/payers/pay_1", false},
		{"Payers.Update", "", func() error { _, err := client.Payers.Update(ctx, "pay_1", &PayerCreateParams{}); return err },
			"POST", "/api/v1/payers/update/pay_1", false},
		{"Payers.Delete", "", func() error { return client.Payers.Delete(ctx, "pay_1") },
			"POST", "/api/v1/payers/delete/pay_1", false},
		{"Payers.List", "", func() error { _, err := client.Payers.List(ctx, nil); return err },
			"GET", "/api/v1/payers", false},
		{"Payers.Validate", "", func() error { _, err := client.Payers.Validate(ctx, &PayerCreateParams{}); return err },
			"POST", "/api/v1/payers/validate", false},

		// Conversions, rates, FX quotes, amendments
		{"Conversions.Create", "", func() error {
			_, err := client.Conversions.Create(ctx, &ConversionCreateParams{TermAgreement: true})
			return err
		}, "POST", "/api/v1/conversions/create", true},
		{"Conversions.Retrieve", "", func() error { _, err := client.Conversions.Retrieve(ctx, "con_1"); return err },
			"GET", "/api/v1/conversions/con_1", false},
		{"Conversions.List", "", func() error { _, err := client.Conversions.List(ctx, nil); return err },
			"GET", "/api/v1/conversions", false},
		{"Rates.Current", "", func() error {
			_, err := client.Rates.Current(ctx, &RateCurrentParams{BuyCurrency: "USD", SellCurrency: "SGD"})
			return err
		}, "GET", "/api/v1/fx/rates/current", false},
		{"FxQuotes.Create", "", func() error { _, err := client.FxQuotes.Create(ctx, &FxQuoteCreateParams{}); return err },
			"POST", "/api/v1/fx/quotes/create", true},
		{"FxQuotes.Retrieve", "", func() error { _, err := client.FxQuotes.Retrieve(ctx, "quo_1"); return err },
			"GET", "/api/v1/fx/quotes/quo_1", false},
		{"ConversionAmendments.Create", "", func() error {
			_, err := client.ConversionAmendments.Create(ctx, &ConversionAmendmentCreateParams{ConversionID: "con_1"})
			return err
		}, "POST", "/api/v1/conversion_amendments/create", true},
		{"ConversionAmendments.Quote", "", func() error {
			_, err := client.ConversionAmendments.Quote(ctx, &ConversionAmendmentCreateParams{ConversionID: "con_1"})
			return err
		}, "POST", "/api/v1/conversion_amendments/quote", true},
		{"ConversionAmendments.Retrieve", "", func() error {
			_, err := client.ConversionAmendments.Retrieve(ctx, "ca_1")
			return err
		}, "GET", "/api/v1/conversion_amendments/ca_1", false},
		{"ConversionAmendments.List", "", func() error {
			_, err := client.ConversionAmendments.List(ctx, &ConversionAmendmentListParams{ConversionID: "con_1"})
			return err
		}, "GET", "/api/v1/conversion_amendments", false},

		// Global accounts, deposits
		{"GlobalAccounts.Create", "", func() error {
			_, err := client.GlobalAccounts.Create(ctx, &GlobalAccountCreateParams{Currency: "SGD"})
			return err
		}, "POST", "/api/v1/global_accounts/create", true},
		{"GlobalAccounts.Retrieve", "", func() error { _, err := client.GlobalAccounts.Retrieve(ctx, "ga_1"); return err },
			"GET", "/api/v1/global_accounts/ga_1", false},
		{"GlobalAccounts.Update", "", func() error {
			_, err := client.GlobalAccounts.Update(ctx, "ga_1", &GlobalAccountUpdateParams{NickName: "ops"})
			return err
		}, "POST", "/api/v1/global_accounts/update/ga_1", false},
		{"GlobalAccounts.Close", "", func() error { _, err := client.GlobalAccounts.Close(ctx, "ga_1"); return err },
			"POST", "/api/v1/global_accounts/ga_1/close", false},
		{"GlobalAccounts.List", "", func() error { _, err := client.GlobalAccounts.List(ctx, nil); return err },
			"GET", "/api/v1/global_accounts", false},
		{"GlobalAccounts.Transactions", "", func() error {
			_, err := client.GlobalAccounts.Transactions(ctx, "ga_1", nil)
			return err
		}, "GET", "/api/v1/global_accounts/ga_1/transactions", false},
		{"Deposits.List", "", func() error { _, err := client.Deposits.List(ctx, nil); return err },
			"GET", "/api/v1/deposits", false},

		// Payment acceptance
		{"PaymentIntents.Create", "", func() error {
			_, err := client.PaymentIntents.Create(ctx, &PaymentIntentCreateParams{Amount: 25, Currency: "USD"})
			return err
		}, "POST", "/api/v1/pa/payment_intents/create", true},
		{"PaymentIntents.Retrieve", "", func() error { _, err := client.PaymentIntents.Retrieve(ctx, "int_1"); return err },
			"GET", "/api/v1/pa/payment_intents/int_1", false},
		{"PaymentIntents.List", "", func() error { _, err := client.PaymentIntents.List(ctx, nil); return err },
			"GET", "/api/v1/pa/payment_intents", false},
		{"PaymentIntents.Confirm", "", func() error {
			_, err := client.PaymentIntents.Confirm(ctx, "int_1", &PaymentIntentActionParams{})
			return err
		}, "POST", "/api/v1/pa/payment_intents/int_1/confirm", false},
		{"PaymentIntents.ConfirmContinue", "", func() error {
			_, err := client.PaymentIntents.ConfirmContinue(ctx, "int_1", &PaymentIntentActionParams{})
			return err
		}, "POST", "/api/v1/pa/payment_intents/int_1/confirm_continue", false},
		{"PaymentIntents.Capture", "", func() error {
			_, err := client.PaymentIntents.Capture(ctx, "int_1", &PaymentIntentActionParams{Amount: 10})
			return err
		}, "POST", "/api/v1/pa/payment_intents/int_1/capture", false},
		{"PaymentIntents.Cancel", "", func() error {
			_, err := client.PaymentIntents.Cancel(ctx, "int_1", &PaymentIntentActionParams{})
			return err
		}, "POST", "/api/v1/pa/payment_intents/int_1/cancel", false},
		{"Customers.Create", "", func() error { _, err := client.Customers.Create(ctx, &CustomerCreateParams{}); return err },
			"POST", "/api/v1/pa/customers/create", true},
		{"Customers.Retrieve", "", func() error { _, err := client.Customers.Retrieve(ctx, "cus_1"); return err },
			"GET", "/api/v1/pa/customers/cus_1", false},
		{"Customers.Update", "", func() error {
			_, err := client.Customers.Update(ctx, "cus_1", &CustomerCreateParams{})
			return err
		}, "POST", "/api/v1/pa/customers/cus_1/update", false},
		{"Customers.List", "", func() error { _, err := client.Customers.List(ctx, nil); return err },
			"GET", "/api/v1/pa/customers", false},
		{"Customers.GenerateClientSecret", "", func() error {
			_, err := client.Customers.GenerateClientSecret(ctx, "cus_1")
			return err
		}, "GET", "/api/v1/pa/customers/cus_1/generate_client_secret", false},
		{"Refunds.Create", "", func() error {
			_, err := client.Refunds.Create(ctx, &RefundCreateParams{PaymentIntentID: "int_1"})
			return err
		}, "POST", "/api/v1/pa/refunds/create", true},
		{"Refunds.Retrieve", "", func() error { _, err := client.Refunds.Retrieve(ctx, "ref_1"); return err },
			"GET", "/api/v1/pa/refunds/ref_1", false},
		{"Refunds.List", "", func() error { _, err := client.Refunds.List(ctx, nil); return err },
			"GET", "/api/v1/pa/refunds", false},

		// Issuing (cardholders have no request_id; cards do)
		{"IssuingCardholders.Create", "", func() error {
			_, err := client.IssuingCardholders.Create(ctx, &CardholderCreateParams{Email: "a@b.c"})
			return err
		}, "POST", "/api/v1/issuing/cardholders/create", false},
		{"IssuingCardholders.Retrieve", "", func() error {
			_, err := client.IssuingCardholders.Retrieve(ctx, "chold_1")
			return err
		}, "GET", "/api/v1/issuing/cardholders/chold_1", false},
		{"IssuingCardholders.Update", "", func() error {
			_, err := client.IssuingCardholders.Update(ctx, "chold_1", &CardholderCreateParams{})
			return err
		}, "POST", "/api/v1/issuing/cardholders/chold_1/update", false},
		{"IssuingCardholders.Delete", "", func() error {
			_, err := client.IssuingCardholders.Delete(ctx, "chold_1")
			return err
		}, "POST", "/api/v1/issuing/cardholders/chold_1/delete", false},
		{"IssuingCardholders.List", "", func() error { _, err := client.IssuingCardholders.List(ctx, nil); return err },
			"GET", "/api/v1/issuing/cardholders", false},
		{"IssuingCards.Create", "", func() error {
			_, err := client.IssuingCards.Create(ctx, &CardCreateParams{CardholderID: "chold_1"})
			return err
		}, "POST", "/api/v1/issuing/cards/create", true},
		{"IssuingCards.Retrieve", "", func() error { _, err := client.IssuingCards.Retrieve(ctx, evilID); return err },
			"GET", "/api/v1/issuing/cards/" + escapedEvilID, false},
		{"IssuingCards.Update", "", func() error {
			_, err := client.IssuingCards.Update(ctx, "card_1", &CardCreateParams{})
			return err
		}, "POST", "/api/v1/issuing/cards/card_1/update", false},
		{"IssuingCards.Activate", "", func() error { return client.IssuingCards.Activate(ctx, "card_1") },
			"POST", "/api/v1/issuing/cards/card_1/activate", false},
		{"IssuingCards.Limits", "", func() error { _, err := client.IssuingCards.Limits(ctx, "card_1"); return err },
			"GET", "/api/v1/issuing/cards/card_1/limits", false},
		{"IssuingCards.List", "", func() error { _, err := client.IssuingCards.List(ctx, nil); return err },
			"GET", "/api/v1/issuing/cards", false},
		{"IssuingTransactions.List", "", func() error { _, err := client.IssuingTransactions.List(ctx, nil); return err },
			"GET", "/api/v1/issuing/transactions", false},
		{"IssuingTransactions.Retrieve", "", func() error {
			_, err := client.IssuingTransactions.Retrieve(ctx, "txn_1")
			return err
		}, "GET", "/api/v1/issuing/transactions/txn_1", false},
		{"IssuingAuthorizations.List", "", func() error {
			_, err := client.IssuingAuthorizations.List(ctx, nil)
			return err
		}, "GET", "/api/v1/issuing/authorizations", false},
		{"IssuingAuthorizations.Retrieve", "", func() error {
			_, err := client.IssuingAuthorizations.Retrieve(ctx, "auth_1")
			return err
		}, "GET", "/api/v1/issuing/authorizations/auth_1", false},

		// Finance, reference, webhooks, simulation
		{"FinancialTransactions.List", "", func() error {
			_, err := client.FinancialTransactions.List(ctx, nil)
			return err
		}, "GET", "/api/v1/pa/financial/transactions", false},
		{"FinancialTransactions.Retrieve", "", func() error {
			_, err := client.FinancialTransactions.Retrieve(ctx, "ft_1")
			return err
		}, "GET", "/api/v1/pa/financial/transactions/ft_1", false},
		{"Settlements.List", "", func() error { _, err := client.Settlements.List(ctx, nil); return err },
			"GET", "/api/v1/pa/financial/settlements", false},
		{"Settlements.Retrieve", "", func() error { _, err := client.Settlements.Retrieve(ctx, "set_1"); return err },
			"GET", "/api/v1/pa/financial/settlements/set_1", false},
		{"Reference.SupportedCurrencies", "", func() error {
			_, err := client.Reference.SupportedCurrencies(ctx)
			return err
		}, "GET", "/api/v1/reference/supported_currencies", false},
		{"Reference.SettlementAccounts", "", func() error {
			_, err := client.Reference.SettlementAccounts(ctx, &ReferenceSettlementAccountsParams{Currency: "SGD"})
			return err
		}, "GET", "/api/v1/reference/settlement_accounts", false},
		{"Reference.InvalidConversionDates", "", func() error {
			_, err := client.Reference.InvalidConversionDates(ctx, "USDSGD")
			return err
		}, "GET", "/api/v1/reference/invalid_conversion_dates", false},
		{"WebhookEndpoints.Create", "", func() error {
			_, err := client.WebhookEndpoints.Create(ctx, &WebhookEndpointCreateParams{
				URL: "https://example.com/hook", Events: []string{"transfer.settled"}})
			return err
		}, "POST", "/api/v1/webhooks/create", true},
		{"WebhookEndpoints.Retrieve", "", func() error { _, err := client.WebhookEndpoints.Retrieve(ctx, "wh_1"); return err },
			"GET", "/api/v1/webhooks/wh_1", false},
		{"WebhookEndpoints.Update", "", func() error {
			_, err := client.WebhookEndpoints.Update(ctx, "wh_1", &WebhookEndpointCreateParams{})
			return err
		}, "POST", "/api/v1/webhooks/wh_1/update", false},
		{"WebhookEndpoints.Delete", "", func() error { return client.WebhookEndpoints.Delete(ctx, "wh_1") },
			"POST", "/api/v1/webhooks/wh_1/delete", false},
		{"WebhookEndpoints.List", "", func() error { _, err := client.WebhookEndpoints.List(ctx, nil); return err },
			"GET", "/api/v1/webhooks", false},
		{"Simulation.CreateDeposit", "", func() error {
			_, err := client.Simulation.CreateDeposit(ctx, &SimulationDepositParams{Amount: 1000, Currency: "USD"})
			return err
		}, "POST", "/api/v1/simulation/deposit/create", false},
		{"Simulation.SettleDeposit", "", func() error { _, err := client.Simulation.SettleDeposit(ctx, "dep_1"); return err },
			"POST", "/api/v1/simulation/deposits/dep_1/settle", false},
		{"Simulation.RejectDeposit", "", func() error { _, err := client.Simulation.RejectDeposit(ctx, "dep_1"); return err },
			"POST", "/api/v1/simulation/deposits/dep_1/reject", false},
		{"Simulation.ReverseDeposit", "", func() error { _, err := client.Simulation.ReverseDeposit(ctx, "dep_1"); return err },
			"POST", "/api/v1/simulation/deposits/dep_1/reverse", false},
		{"Simulation.TransitionTransfer", "", func() error {
			_, err := client.Simulation.TransitionTransfer(ctx, "tra_1", &SimulationTransitionParams{NextStatus: "PAID"})
			return err
		}, "POST", "/api/v1/simulation/transfers/tra_1/transition", false},
		{"Simulation.TransitionPayment", "", func() error {
			_, err := client.Simulation.TransitionPayment(ctx, "pmt_1", &SimulationTransitionParams{NextStatus: "SETTLED"})
			return err
		}, "POST", "/api/v1/simulation/payments/pmt_1/transition", false},
	}
}

func TestServiceRouting(t *testing.T) {
	server, rec := newRecordingServer(t)
	client := mustClient(t, WithBaseURL(server.URL))
	for _, tt := range buildRoutingTests(context.Background(), client) {
		t.Run(tt.name, func(t *testing.T) {
			rec.response = tt.response
			if err := tt.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tt.wantMethod {
				t.Errorf("method = %s, want %s", rec.method, tt.wantMethod)
			}
			if rec.path != tt.wantPath {
				t.Errorf("path = %s, want %s", rec.path, tt.wantPath)
			}
			if tt.wantReqID {
				id, _ := rec.body["request_id"].(string)
				if id == "" {
					t.Errorf("request_id missing from body: %s", rec.rawBody)
				}
			}
		})
	}
}

// TestServiceMethodsPropagateErrors drives every wrapped method against a
// server that always fails, asserting each surfaces a typed *Error.
func TestServiceMethodsPropagateErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == loginPath {
			loginOK(w)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"code":"forbidden","message":"nope"}`)
	}))
	t.Cleanup(server.Close)
	client := mustClient(t, WithBaseURL(server.URL))
	for _, tt := range buildRoutingTests(context.Background(), client) {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
				t.Fatalf("err = %v, want 403 *Error", err)
			}
		})
	}
}

// TestAllIterators exercises every range-over-func iterator once.
func TestAllIterators(t *testing.T) {
	server, rec := newRecordingServer(t)
	rec.response = `{"has_more":false,"items":[{}]}`
	client := mustClient(t, WithBaseURL(server.URL))
	ctx := context.Background()

	iterate := func(name string, seq func(yield func() bool)) {
		count := 0
		seq(func() bool { count++; return true })
		if count != 1 {
			t.Errorf("%s yielded %d items, want 1", name, count)
		}
	}
	iterate("Transfers.All", func(y func() bool) {
		for _, err := range client.Transfers.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("BatchTransfers.All", func(y func() bool) {
		for _, err := range client.BatchTransfers.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("BatchTransfers.AllItems", func(y func() bool) {
		for _, err := range client.BatchTransfers.AllItems(ctx, "bat_1", nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("WalletTransfers.All", func(y func() bool) {
		for _, err := range client.WalletTransfers.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Payers.All", func(y func() bool) {
		for _, err := range client.Payers.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Conversions.All", func(y func() bool) {
		for _, err := range client.Conversions.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("ConversionAmendments.All", func(y func() bool) {
		for _, err := range client.ConversionAmendments.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("GlobalAccounts.All", func(y func() bool) {
		for _, err := range client.GlobalAccounts.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("GlobalAccounts.AllTransactions", func(y func() bool) {
		for _, err := range client.GlobalAccounts.AllTransactions(ctx, "ga_1", nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Deposits.All", func(y func() bool) {
		for _, err := range client.Deposits.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Balances.AllHistory", func(y func() bool) {
		for _, err := range client.Balances.AllHistory(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Customers.All", func(y func() bool) {
		for _, err := range client.Customers.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("PaymentIntents.All", func(y func() bool) {
		for _, err := range client.PaymentIntents.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Refunds.All", func(y func() bool) {
		for _, err := range client.Refunds.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("IssuingCardholders.All", func(y func() bool) {
		for _, err := range client.IssuingCardholders.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("IssuingCards.All", func(y func() bool) {
		for _, err := range client.IssuingCards.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("IssuingTransactions.All", func(y func() bool) {
		for _, err := range client.IssuingTransactions.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("IssuingAuthorizations.All", func(y func() bool) {
		for _, err := range client.IssuingAuthorizations.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("FinancialTransactions.All", func(y func() bool) {
		for _, err := range client.FinancialTransactions.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("Settlements.All", func(y func() bool) {
		for _, err := range client.Settlements.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
	iterate("WebhookEndpoints.All", func(y func() bool) {
		for _, err := range client.WebhookEndpoints.All(ctx, nil) {
			if err != nil {
				t.Fatal(err)
			}
			if !y() {
				return
			}
		}
	})
}

func TestPageAllIterator(t *testing.T) {
	server, rec := newRecordingServer(t)
	rec.response = `{"has_more":false,"items":[{"id":"tra_1"}]}`
	client := mustClient(t, WithBaseURL(server.URL))
	page, err := client.Transfers.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	count := 0
	for transfer, err := range page.All(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if transfer.ID != "tra_1" {
			t.Fatalf("transfer = %+v", transfer)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
}

func TestExtraParamsMergedIntoBody(t *testing.T) {
	server, rec := newRecordingServer(t)
	client := mustClient(t, WithBaseURL(server.URL))
	_, err := client.Transfers.Create(context.Background(), &TransferCreateParams{
		Params:    Params{ExtraParams: map[string]any{"brand_new_flag": true, "reference": "overridden"}},
		Reference: "typed",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.body["brand_new_flag"] != true {
		t.Errorf("extra param missing: %s", rec.rawBody)
	}
	if rec.body["reference"] != "overridden" {
		t.Errorf("extra params should override typed fields: %s", rec.rawBody)
	}
}

func TestRateCurrentQueryEncoding(t *testing.T) {
	server, rec := newRecordingServer(t)
	client := mustClient(t, WithBaseURL(server.URL))
	_, err := client.Rates.Current(context.Background(), &RateCurrentParams{
		BuyCurrency:  "USD",
		SellCurrency: "SGD",
		BuyAmount:    1234.56,
	})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	for _, want := range []string{"buy_currency=USD", "sell_currency=SGD", "buy_amount=1234.56"} {
		if !containsQuery(rec.query, want) {
			t.Errorf("query %q missing %q", rec.query, want)
		}
	}
	if containsQuery(rec.query, "sell_amount") {
		t.Errorf("zero sell_amount must be omitted: %q", rec.query)
	}
}

func containsQuery(query, fragment string) bool {
	return strings.Contains("&"+query+"&", "&"+fragment) // fragment may be a bare key
}

func TestResponseRawPreservedOnRetrieve(t *testing.T) {
	server, rec := newRecordingServer(t)
	rec.response = `{"id":"tra_1","status":"PAID","field_from_2027":"preserved"}`
	client := mustClient(t, WithBaseURL(server.URL))
	transfer, err := client.Transfers.Retrieve(context.Background(), "tra_1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if string(transfer.Raw) != rec.response {
		t.Fatalf("Raw = %s, want full body", transfer.Raw)
	}
	var raw map[string]any
	if err := json.Unmarshal(transfer.Raw, &raw); err != nil {
		t.Fatalf("Raw is not valid JSON: %v", err)
	}
	if raw["field_from_2027"] != "preserved" {
		t.Fatal("unknown field lost")
	}
}

func TestNewRequestIDShape(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := newRequestID()
		if len(id) != 36 || id[14] != '4' {
			t.Fatalf("newRequestID() = %q, want UUIDv4 shape", id)
		}
		if seen[id] {
			t.Fatalf("duplicate request id generated: %s", id)
		}
		seen[id] = true
	}
}
