package airwallex

import (
	"context"
	"iter"
)

const (
	issuingTransactionsBasePath   = "/api/v1/issuing/transactions"
	issuingAuthorizationsBasePath = "/api/v1/issuing/authorizations"
)

// IssuingTransaction is a cleared card transaction
// (/api/v1/issuing/transactions).
type IssuingTransaction struct {
	APIResource
	TransactionID   string `json:"transaction_id"`
	TransactionType string `json:"transaction_type"`
	Status          string `json:"status"`

	CardID               string `json:"card_id"`
	CardNickname         string `json:"card_nickname"`
	MaskedCardNumber     string `json:"masked_card_number"`
	DigitalWalletTokenID string `json:"digital_wallet_token_id"`

	TransactionAmount   float64          `json:"transaction_amount"`
	TransactionCurrency string           `json:"transaction_currency"`
	BillingAmount       float64          `json:"billing_amount"`
	BillingCurrency     string           `json:"billing_currency"`
	FeeDetails          []map[string]any `json:"fee_details"`

	Merchant                       map[string]any `json:"merchant"`
	AcquiringInstitutionIdentifier string         `json:"acquiring_institution_identifier"`
	AuthCode                       string         `json:"auth_code"`
	NetworkTransactionID           string         `json:"network_transaction_id"`
	RetrievalRef                   string         `json:"retrieval_ref"`
	LifecycleID                    string         `json:"lifecycle_id"`
	MatchedAuthorizations          []string       `json:"matched_authorizations"`

	RiskDetails   map[string]any `json:"risk_details"`
	FailureReason string         `json:"failure_reason"`
	ClientData    string         `json:"client_data"`

	TransactionDate string `json:"transaction_date"`
	PostedDate      string `json:"posted_date"`
}

// IssuingAuthorization is a card authorization
// (/api/v1/issuing/authorizations).
type IssuingAuthorization struct {
	APIResource
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`

	CardID               string `json:"card_id"`
	CardNickname         string `json:"card_nickname"`
	MaskedCardNumber     string `json:"masked_card_number"`
	DigitalWalletTokenID string `json:"digital_wallet_token_id"`

	TransactionAmount   float64          `json:"transaction_amount"`
	TransactionCurrency string           `json:"transaction_currency"`
	BillingAmount       float64          `json:"billing_amount"`
	BillingCurrency     string           `json:"billing_currency"`
	FeeDetails          []map[string]any `json:"fee_details"`

	Merchant                       map[string]any `json:"merchant"`
	AcquiringInstitutionIdentifier string         `json:"acquiring_institution_identifier"`
	AuthCode                       string         `json:"auth_code"`
	NetworkTransactionID           string         `json:"network_transaction_id"`
	RetrievalRef                   string         `json:"retrieval_ref"`
	LifecycleID                    string         `json:"lifecycle_id"`
	UpdatedByTransaction           string         `json:"updated_by_transaction"`

	RiskDetails   map[string]any `json:"risk_details"`
	FailureReason string         `json:"failure_reason"`
	ClientData    string         `json:"client_data"`

	CreateTime string `json:"create_time"`
	ExpiryDate string `json:"expiry_date"`
}

// IssuingTransactionListParams filter IssuingTransactionsService.List.
type IssuingTransactionListParams struct {
	ListParams
	CardID               string `json:"card_id,omitempty"`
	BillingCurrency      string `json:"billing_currency,omitempty"`
	TransactionType      string `json:"transaction_type,omitempty"`
	DigitalWalletTokenID string `json:"digital_wallet_token_id,omitempty"`
	LifecycleID          string `json:"lifecycle_id,omitempty"`
	RetrievalRef         string `json:"retrieval_ref,omitempty"`
	FromCreatedAt        string `json:"from_created_at,omitempty"`
	ToCreatedAt          string `json:"to_created_at,omitempty"`
}

// IssuingAuthorizationListParams filter IssuingAuthorizationsService.List.
type IssuingAuthorizationListParams struct {
	ListParams
	CardID               string `json:"card_id,omitempty"`
	Status               string `json:"status,omitempty"`
	BillingCurrency      string `json:"billing_currency,omitempty"`
	DigitalWalletTokenID string `json:"digital_wallet_token_id,omitempty"`
	LifecycleID          string `json:"lifecycle_id,omitempty"`
	RetrievalRef         string `json:"retrieval_ref,omitempty"`
	FromCreatedAt        string `json:"from_created_at,omitempty"`
	ToCreatedAt          string `json:"to_created_at,omitempty"`
}

// IssuingTransactionsService lists cleared card transactions.
type IssuingTransactionsService struct{ client *Client }

// List returns one page of card transactions, filtered by params (may be
// nil).
func (s *IssuingTransactionsService) List(ctx context.Context, params *IssuingTransactionListParams) (*Page[IssuingTransaction], error) {
	return listPage[IssuingTransaction](ctx, s.client, issuingTransactionsBasePath, params)
}

// All iterates every card transaction across every page, fetching lazily.
func (s *IssuingTransactionsService) All(ctx context.Context, params *IssuingTransactionListParams) iter.Seq2[IssuingTransaction, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Retrieve fetches a single card transaction by id.
func (s *IssuingTransactionsService) Retrieve(ctx context.Context, transactionID string) (*IssuingTransaction, error) {
	transaction := &IssuingTransaction{}
	if err := s.client.get(ctx, issuingTransactionsBasePath+"/"+pathEscape(transactionID), nil, transaction); err != nil {
		return nil, err
	}
	return transaction, nil
}

// IssuingAuthorizationsService lists card authorizations.
type IssuingAuthorizationsService struct{ client *Client }

// List returns one page of authorizations, filtered by params (may be
// nil).
func (s *IssuingAuthorizationsService) List(ctx context.Context, params *IssuingAuthorizationListParams) (*Page[IssuingAuthorization], error) {
	return listPage[IssuingAuthorization](ctx, s.client, issuingAuthorizationsBasePath, params)
}

// All iterates every authorization across every page, fetching lazily.
func (s *IssuingAuthorizationsService) All(ctx context.Context, params *IssuingAuthorizationListParams) iter.Seq2[IssuingAuthorization, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Retrieve fetches a single authorization by id.
func (s *IssuingAuthorizationsService) Retrieve(ctx context.Context, authorizationID string) (*IssuingAuthorization, error) {
	authorization := &IssuingAuthorization{}
	if err := s.client.get(ctx, issuingAuthorizationsBasePath+"/"+pathEscape(authorizationID), nil, authorization); err != nil {
		return nil, err
	}
	return authorization, nil
}
