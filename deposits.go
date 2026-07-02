package airwallex

import (
	"context"
	"iter"
)

const depositsBasePath = "/api/v1/deposits"

// Deposit is money received into the wallet (GET /api/v1/deposits).
type Deposit struct {
	APIResource
	ID                    string         `json:"id"`
	Amount                float64        `json:"amount"`
	Currency              string         `json:"currency"`
	Status                string         `json:"status"`
	Reference             string         `json:"reference"`
	Payer                 map[string]any `json:"payer"`
	Fee                   map[string]any `json:"fee"`
	FundingSourceID       string         `json:"funding_source_id"`
	GlobalAccountID       string         `json:"global_account_id"`
	ProviderTransactionID string         `json:"provider_transaction_id"`
	EstimatedSettledAt    string         `json:"estimated_settled_at"`
	CreatedAt             string         `json:"created_at"`
}

// DepositListParams filter DepositsService.List.
type DepositListParams struct {
	ListParams
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// DepositsService lists deposits received into the wallet.
type DepositsService struct{ client *Client }

// List returns one page of deposits, filtered by params (may be nil).
func (s *DepositsService) List(ctx context.Context, params *DepositListParams) (*Page[Deposit], error) {
	return listPage[Deposit](ctx, s.client, depositsBasePath, params)
}

// All iterates every deposit across every page, fetching lazily.
func (s *DepositsService) All(ctx context.Context, params *DepositListParams) iter.Seq2[Deposit, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
