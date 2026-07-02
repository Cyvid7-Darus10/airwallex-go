package airwallex

import (
	"context"
	"encoding/json"
	"iter"
)

const (
	balancesCurrentPath = "/api/v1/balances/current"
	balancesHistoryPath = "/api/v1/balances/history"
)

// Balance is a wallet balance in one currency
// (GET /api/v1/balances/current).
type Balance struct {
	APIResource
	Currency        string  `json:"currency"`
	AvailableAmount float64 `json:"available_amount"`
	PendingAmount   float64 `json:"pending_amount"`
	ReservedAmount  float64 `json:"reserved_amount"`
	TotalAmount     float64 `json:"total_amount"`
}

// BalanceHistoryItem is one ledger movement
// (GET /api/v1/balances/history).
type BalanceHistoryItem struct {
	APIResource
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	Balance     float64 `json:"balance"`
	Fee         float64 `json:"fee"`
	Description string  `json:"description"`
	Source      string  `json:"source"`
	SourceType  string  `json:"source_type"`
	PostedAt    string  `json:"posted_at"`
}

// BalanceHistoryParams filter BalancesService.History.
type BalanceHistoryParams struct {
	ListParams
	Currency   string `json:"currency,omitempty"`
	FromPostAt string `json:"from_post_at,omitempty"`
	ToPostAt   string `json:"to_post_at,omitempty"`
}

// BalancesService reports current and historical wallet balances.
type BalancesService struct{ client *Client }

// Current returns the wallet balance in every currency.
func (s *BalancesService) Current(ctx context.Context) ([]Balance, error) {
	var raws []json.RawMessage
	if err := s.client.get(ctx, balancesCurrentPath, nil, &raws); err != nil {
		return nil, err
	}
	return decodeItems[Balance](raws)
}

// History returns one page of ledger movements, filtered by params (may be
// nil).
func (s *BalancesService) History(ctx context.Context, params *BalanceHistoryParams) (*Page[BalanceHistoryItem], error) {
	return listPage[BalanceHistoryItem](ctx, s.client, balancesHistoryPath, params)
}

// AllHistory iterates every ledger movement across every page.
func (s *BalancesService) AllHistory(ctx context.Context, params *BalanceHistoryParams) iter.Seq2[BalanceHistoryItem, error] {
	page, err := s.History(ctx, params)
	return iterPages(ctx, page, err)
}
