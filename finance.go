package airwallex

import (
	"context"
	"iter"
)

const (
	financialTransactionsBasePath = "/api/v1/pa/financial/transactions"
	settlementsBasePath           = "/api/v1/pa/financial/settlements"
)

// FinancialTransaction is one payment-acceptance ledger entry
// (/api/v1/pa/financial/transactions).
type FinancialTransaction struct {
	APIResource
	ID              string  `json:"id"`
	BatchID         string  `json:"batch_id"`
	SourceID        string  `json:"source_id"`
	FundingSourceID string  `json:"funding_source_id"`
	SourceType      string  `json:"source_type"`
	TransactionType string  `json:"transaction_type"`
	Currency        string  `json:"currency"`
	Amount          float64 `json:"amount"`
	Net             float64 `json:"net"`
	Fee             float64 `json:"fee"`
	ClientRate      float64 `json:"client_rate"`
	CurrencyPair    string  `json:"currency_pair"`
	Description     string  `json:"description"`
	Status          string  `json:"status"`

	EstimatedSettledAt string `json:"estimated_settled_at"`
	SettledAt          string `json:"settled_at"`
	CreatedAt          string `json:"created_at"`
}

// Settlement is one payment-acceptance settlement
// (/api/v1/pa/financial/settlements).
type Settlement struct {
	APIResource
	ID       string  `json:"id"`
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Fee      float64 `json:"fee"`
	Status   string  `json:"status"`

	EstimatedSettledAt string `json:"estimated_settled_at"`
	SettledAt          string `json:"settled_at"`
	CreatedAt          string `json:"created_at"`
}

// FinancialTransactionListParams filter
// FinancialTransactionsService.List.
type FinancialTransactionListParams struct {
	ListParams
	BatchID       string `json:"batch_id,omitempty"`
	Currency      string `json:"currency,omitempty"`
	SourceID      string `json:"source_id,omitempty"`
	Status        string `json:"status,omitempty"`
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// SettlementListParams filter SettlementsService.List.
type SettlementListParams struct {
	ListParams
	Currency      string `json:"currency,omitempty"`
	Status        string `json:"status,omitempty"`
	FromSettledAt string `json:"from_settled_at,omitempty"`
	ToSettledAt   string `json:"to_settled_at,omitempty"`
}

// FinancialTransactionsService lists payment-acceptance ledger activity.
type FinancialTransactionsService struct{ client *Client }

// List returns one page of financial transactions, filtered by params (may
// be nil).
func (s *FinancialTransactionsService) List(ctx context.Context, params *FinancialTransactionListParams) (*Page[FinancialTransaction], error) {
	return listPage[FinancialTransaction](ctx, s.client, financialTransactionsBasePath, params)
}

// All iterates every financial transaction across every page.
func (s *FinancialTransactionsService) All(ctx context.Context, params *FinancialTransactionListParams) iter.Seq2[FinancialTransaction, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Retrieve fetches a single financial transaction by id.
func (s *FinancialTransactionsService) Retrieve(ctx context.Context, transactionID string) (*FinancialTransaction, error) {
	transaction := &FinancialTransaction{}
	if err := s.client.get(ctx, financialTransactionsBasePath+"/"+pathEscape(transactionID), nil, transaction); err != nil {
		return nil, err
	}
	return transaction, nil
}

// SettlementsService lists payment-acceptance settlements.
type SettlementsService struct{ client *Client }

// List returns one page of settlements, filtered by params (may be nil).
func (s *SettlementsService) List(ctx context.Context, params *SettlementListParams) (*Page[Settlement], error) {
	return listPage[Settlement](ctx, s.client, settlementsBasePath, params)
}

// All iterates every settlement across every page, fetching lazily.
func (s *SettlementsService) All(ctx context.Context, params *SettlementListParams) iter.Seq2[Settlement, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Retrieve fetches a single settlement by id.
func (s *SettlementsService) Retrieve(ctx context.Context, settlementID string) (*Settlement, error) {
	settlement := &Settlement{}
	if err := s.client.get(ctx, settlementsBasePath+"/"+pathEscape(settlementID), nil, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}
