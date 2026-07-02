package airwallex

import (
	"context"
	"encoding/json"
	"iter"
)

const transfersBasePath = "/api/v1/transfers"

// Transfer is a payout to a beneficiary (/api/v1/transfers).
type Transfer struct {
	APIResource
	ID               string `json:"id"`
	RequestID        string `json:"request_id"`
	Status           string `json:"status"`
	ShortReferenceID string `json:"short_reference_id"`

	SourceAmount     float64 `json:"source_amount"`
	SourceCurrency   string  `json:"source_currency"`
	TransferAmount   float64 `json:"transfer_amount"`
	TransferCurrency string  `json:"transfer_currency"`
	TransferMethod   string  `json:"transfer_method"`
	TransferDate     string  `json:"transfer_date"`

	AmountBeneficiaryReceives float64 `json:"amount_beneficiary_receives"`
	AmountPayerPays           float64 `json:"amount_payer_pays"`
	FeeAmount                 float64 `json:"fee_amount"`
	FeeCurrency               string  `json:"fee_currency"`
	FeePaidBy                 string  `json:"fee_paid_by"`
	SwiftChargeOption         string  `json:"swift_charge_option"`

	Beneficiary   *BeneficiaryDetails `json:"beneficiary"`
	BeneficiaryID string              `json:"beneficiary_id"`
	Payer         map[string]any      `json:"payer"`

	Reference string         `json:"reference"`
	Reason    string         `json:"reason"`
	Remarks   string         `json:"remarks"`
	Metadata  map[string]any `json:"metadata"`

	FailureReason string `json:"failure_reason"`
	FailureType   string `json:"failure_type"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TransferCreateParams are the parameters for TransfersService.Create.
type TransferCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID     string              `json:"request_id,omitempty"`
	BeneficiaryID string              `json:"beneficiary_id,omitempty"`
	Beneficiary   *BeneficiaryDetails `json:"beneficiary,omitempty"`
	Payer         map[string]any      `json:"payer,omitempty"`

	SourceCurrency   string  `json:"source_currency,omitempty"`
	SourceAmount     float64 `json:"source_amount,omitempty"`
	TransferAmount   float64 `json:"transfer_amount,omitempty"`
	TransferCurrency string  `json:"transfer_currency,omitempty"`
	TransferMethod   string  `json:"transfer_method,omitempty"`
	TransferDate     string  `json:"transfer_date,omitempty"`

	FeePaidBy         string `json:"fee_paid_by,omitempty"`
	SwiftChargeOption string `json:"swift_charge_option,omitempty"`

	Reference string         `json:"reference,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	Remarks   string         `json:"remarks,omitempty"`
	QuoteID   string         `json:"quote_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TransferListParams filter TransfersService.List.
type TransferListParams struct {
	ListParams
	Status        string `json:"status,omitempty"`
	Currency      string `json:"currency,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// TransferConfirmFundingParams are the parameters for
// TransfersService.ConfirmFunding.
type TransferConfirmFundingParams struct {
	Params
	FundingSourceID string `json:"funding_source_id,omitempty"`
}

// TransfersService creates and manages payouts to beneficiaries.
//
// Requires an API version of 2024-01-31 or later (earlier versions call
// this resource "payments"). Use WithAPIVersion if your account default is
// older.
type TransfersService struct{ client *Client }

// Create creates a payout. A request_id is generated automatically when
// params.RequestID is empty, making the call idempotent — Airwallex never
// executes the same request_id twice, even across the SDK's automatic
// retries.
func (s *TransfersService) Create(ctx context.Context, params *TransferCreateParams) (*Transfer, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	transfer := &Transfer{}
	if err := s.client.post(ctx, transfersBasePath+"/create", body, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// Retrieve fetches a single transfer by id.
func (s *TransfersService) Retrieve(ctx context.Context, transferID string) (*Transfer, error) {
	transfer := &Transfer{}
	if err := s.client.get(ctx, transfersBasePath+"/"+pathEscape(transferID), nil, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// List returns one page of transfers, filtered by params (which may be nil).
func (s *TransfersService) List(ctx context.Context, params *TransferListParams) (*Page[Transfer], error) {
	return listPage[Transfer](ctx, s.client, transfersBasePath, params)
}

// All iterates every transfer across every page, fetching lazily.
func (s *TransfersService) All(ctx context.Context, params *TransferListParams) iter.Seq2[Transfer, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Cancel cancels a transfer that has not yet been dispatched.
func (s *TransfersService) Cancel(ctx context.Context, transferID string) (*Transfer, error) {
	transfer := &Transfer{}
	if err := s.client.post(ctx, transfersBasePath+"/"+pathEscape(transferID)+"/cancel", nil, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// Validate validates a transfer payload without creating it, returning the
// raw validation result from Airwallex.
func (s *TransfersService) Validate(ctx context.Context, params *TransferCreateParams) (json.RawMessage, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := s.client.post(ctx, transfersBasePath+"/validate", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ConfirmFunding confirms funding for a transfer that is awaiting funds:
// once the money has arrived (or you choose the funding source), confirming
// releases the transfer for processing.
func (s *TransfersService) ConfirmFunding(ctx context.Context, transferID string, params *TransferConfirmFundingParams) (*Transfer, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	transfer := &Transfer{}
	path := transfersBasePath + "/" + pathEscape(transferID) + "/confirm_funding"
	if err := s.client.post(ctx, path, body, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}
