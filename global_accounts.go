package airwallex

import (
	"context"
	"iter"
)

const globalAccountsBasePath = "/api/v1/global_accounts"

// GlobalAccount is a local currency account for collecting funds
// (/api/v1/global_accounts).
type GlobalAccount struct {
	APIResource
	ID                 string         `json:"id"`
	RequestID          string         `json:"request_id"`
	AccountName        string         `json:"account_name"`
	AccountNumber      string         `json:"account_number"`
	AccountRoutingType string         `json:"account_routing_type"`
	AccountRoutingVal  string         `json:"account_routing_value"`
	BranchCode         string         `json:"branch_code"`
	ClearingSystems    []string       `json:"clearing_systems"`
	CountryCode        string         `json:"country_code"`
	Currency           string         `json:"currency"`
	InstitutionName    string         `json:"institution_name"`
	NickName           string         `json:"nick_name"`
	PaymentMethods     []string       `json:"payment_methods"`
	Status             string         `json:"status"`
	SwiftCode          string         `json:"swift_code"`
	RegisteredEmail    string         `json:"registered_email"`
	AlternateAccountID map[string]any `json:"alternate_account_identifiers"`
}

// GlobalAccountTransaction is one transaction received into a global
// account.
type GlobalAccountTransaction struct {
	APIResource
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	Fee             float64 `json:"fee"`
	PayerName       string  `json:"payer_name"`
	Reference       string  `json:"reference"`
	Status          string  `json:"status"`
	TransactionDate string  `json:"transaction_date"`
}

// GlobalAccountCreateParams are the parameters for
// GlobalAccountsService.Create.
type GlobalAccountCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID      string   `json:"request_id,omitempty"`
	CountryCode    string   `json:"country_code,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	NickName       string   `json:"nick_name,omitempty"`
	PaymentMethods []string `json:"payment_methods,omitempty"`
}

// GlobalAccountUpdateParams are the parameters for
// GlobalAccountsService.Update.
type GlobalAccountUpdateParams struct {
	Params
	NickName string `json:"nick_name,omitempty"`
}

// GlobalAccountListParams filter GlobalAccountsService.List.
type GlobalAccountListParams struct {
	ListParams
	Currency      string `json:"currency,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	Status        string `json:"status,omitempty"`
	NickName      string `json:"nick_name,omitempty"`
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// GlobalAccountTransactionsParams filter
// GlobalAccountsService.Transactions.
type GlobalAccountTransactionsParams struct {
	ListParams
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// GlobalAccountsService manages local currency accounts for collecting
// funds.
type GlobalAccountsService struct{ client *Client }

// Create opens a global account. A request_id is generated automatically
// when params.RequestID is empty, making the call idempotent.
func (s *GlobalAccountsService) Create(ctx context.Context, params *GlobalAccountCreateParams) (*GlobalAccount, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	account := &GlobalAccount{}
	if err := s.client.post(ctx, globalAccountsBasePath+"/create", body, account); err != nil {
		return nil, err
	}
	return account, nil
}

// Retrieve fetches a single global account by id.
func (s *GlobalAccountsService) Retrieve(ctx context.Context, globalAccountID string) (*GlobalAccount, error) {
	account := &GlobalAccount{}
	if err := s.client.get(ctx, globalAccountsBasePath+"/"+pathEscape(globalAccountID), nil, account); err != nil {
		return nil, err
	}
	return account, nil
}

// Update changes a global account's mutable details.
func (s *GlobalAccountsService) Update(ctx context.Context, globalAccountID string, params *GlobalAccountUpdateParams) (*GlobalAccount, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	account := &GlobalAccount{}
	path := globalAccountsBasePath + "/update/" + pathEscape(globalAccountID)
	if err := s.client.post(ctx, path, body, account); err != nil {
		return nil, err
	}
	return account, nil
}

// Close closes a global account.
func (s *GlobalAccountsService) Close(ctx context.Context, globalAccountID string) (*GlobalAccount, error) {
	account := &GlobalAccount{}
	path := globalAccountsBasePath + "/" + pathEscape(globalAccountID) + "/close"
	if err := s.client.post(ctx, path, nil, account); err != nil {
		return nil, err
	}
	return account, nil
}

// List returns one page of global accounts, filtered by params (may be
// nil).
func (s *GlobalAccountsService) List(ctx context.Context, params *GlobalAccountListParams) (*Page[GlobalAccount], error) {
	return listPage[GlobalAccount](ctx, s.client, globalAccountsBasePath, params)
}

// All iterates every global account across every page, fetching lazily.
func (s *GlobalAccountsService) All(ctx context.Context, params *GlobalAccountListParams) iter.Seq2[GlobalAccount, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Transactions returns one page of transactions received into a global
// account.
func (s *GlobalAccountsService) Transactions(ctx context.Context, globalAccountID string, params *GlobalAccountTransactionsParams) (*Page[GlobalAccountTransaction], error) {
	path := globalAccountsBasePath + "/" + pathEscape(globalAccountID) + "/transactions"
	return listPage[GlobalAccountTransaction](ctx, s.client, path, params)
}

// AllTransactions iterates every transaction across every page.
func (s *GlobalAccountsService) AllTransactions(ctx context.Context, globalAccountID string, params *GlobalAccountTransactionsParams) iter.Seq2[GlobalAccountTransaction, error] {
	page, err := s.Transactions(ctx, globalAccountID, params)
	return iterPages(ctx, page, err)
}
