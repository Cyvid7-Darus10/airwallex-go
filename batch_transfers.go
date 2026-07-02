package airwallex

import (
	"context"
	"iter"
)

const batchTransfersBasePath = "/api/v1/batch_transfers"

// BatchQuoteDetails is one FX quote inside a batch quote summary.
type BatchQuoteDetails struct {
	AmountBeneficiaryReceives float64 `json:"amount_beneficiary_receives"`
	AmountPayerPays           float64 `json:"amount_payer_pays"`
	ClientRate                float64 `json:"client_rate"`
	CurrencyPair              string  `json:"currency_pair"`
	FeeAmount                 float64 `json:"fee_amount"`
	FeeCurrency               string  `json:"fee_currency"`
	PaymentCurrency           string  `json:"payment_currency"`
	SourceCurrency            string  `json:"source_currency"`
}

// BatchQuoteSummary aggregates the FX quotes locked for a batch.
type BatchQuoteSummary struct {
	ExpiresAt    string              `json:"expires_at"`
	LastQuotedAt string              `json:"last_quoted_at"`
	Quotes       []BatchQuoteDetails `json:"quotes"`
	Validity     string              `json:"validity"`
}

// BatchFunding describes how a batch transfer is funded.
type BatchFunding struct {
	DepositType     string         `json:"deposit_type"`
	FailureDetails  map[string]any `json:"failure_details"`
	FailureReason   string         `json:"failure_reason"`
	FundingSourceID string         `json:"funding_source_id"`
	Reference       string         `json:"reference"`
	Status          string         `json:"status"`
}

// BatchTransfer is a batch of payouts (/api/v1/batch_transfers).
type BatchTransfer struct {
	APIResource
	ID               string `json:"id"`
	RequestID        string `json:"request_id"`
	ShortReferenceID string `json:"short_reference_id"`
	Status           string `json:"status"`
	Name             string `json:"name"`
	Remarks          string `json:"remarks"`

	Funding      *BatchFunding      `json:"funding"`
	QuoteSummary *BatchQuoteSummary `json:"quote_summary"`

	TotalItemCount int            `json:"total_item_count"`
	ValidItemCount int            `json:"valid_item_count"`
	TransferDate   string         `json:"transfer_date"`
	Metadata       map[string]any `json:"metadata"`
	UpdatedAt      string         `json:"updated_at"`
}

// BatchTransferItem is one payout inside a batch.
type BatchTransferItem struct {
	APIResource
	ID            string           `json:"id"`
	RequestID     string           `json:"request_id"`
	Status        string           `json:"status"`
	TransferDraft map[string]any   `json:"transfer_draft"`
	TransferID    string           `json:"transfer_id"`
	Errors        []map[string]any `json:"errors"`
	UpdatedAt     string           `json:"updated_at"`
}

// BatchTransferCreateParams are the parameters for
// BatchTransfersService.Create.
type BatchTransferCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID    string           `json:"request_id,omitempty"`
	Name         string           `json:"name,omitempty"`
	Remarks      string           `json:"remarks,omitempty"`
	TransferDate string           `json:"transfer_date,omitempty"`
	Items        []map[string]any `json:"items,omitempty"`
	Funding      map[string]any   `json:"funding,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// BatchTransferListParams filter BatchTransfersService.List.
type BatchTransferListParams struct {
	ListParams
	Status           string `json:"status,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	ShortReferenceID string `json:"short_reference_id,omitempty"`
}

// BatchTransferQuoteParams are the parameters for
// BatchTransfersService.Quote.
type BatchTransferQuoteParams struct {
	Params
}

// BatchTransfersService manages batches of payouts through their full
// lifecycle: create, add/delete items, quote, submit.
type BatchTransfersService struct{ client *Client }

// Create creates a batch transfer. A request_id is generated automatically
// when params.RequestID is empty, making the call idempotent.
func (s *BatchTransfersService) Create(ctx context.Context, params *BatchTransferCreateParams) (*BatchTransfer, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	batch := &BatchTransfer{}
	if err := s.client.post(ctx, batchTransfersBasePath+"/create", body, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// Retrieve fetches a single batch transfer by id.
func (s *BatchTransfersService) Retrieve(ctx context.Context, batchTransferID string) (*BatchTransfer, error) {
	batch := &BatchTransfer{}
	if err := s.client.get(ctx, batchTransfersBasePath+"/"+pathEscape(batchTransferID), nil, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// List returns one page of batch transfers, filtered by params (may be nil).
func (s *BatchTransfersService) List(ctx context.Context, params *BatchTransferListParams) (*Page[BatchTransfer], error) {
	return listPage[BatchTransfer](ctx, s.client, batchTransfersBasePath, params)
}

// All iterates every batch transfer across every page, fetching lazily.
func (s *BatchTransfersService) All(ctx context.Context, params *BatchTransferListParams) iter.Seq2[BatchTransfer, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// AddItems adds transfer drafts to a batch that has not been submitted.
func (s *BatchTransfersService) AddItems(ctx context.Context, batchTransferID string, items []map[string]any) (*BatchTransfer, error) {
	batch := &BatchTransfer{}
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/add_items"
	if err := s.client.post(ctx, path, map[string]any{"items": items}, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// DeleteItems removes items from a batch that has not been submitted.
func (s *BatchTransfersService) DeleteItems(ctx context.Context, batchTransferID string, itemIDs []string) (*BatchTransfer, error) {
	batch := &BatchTransfer{}
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/delete_items"
	if err := s.client.post(ctx, path, map[string]any{"item_ids": itemIDs}, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// Items returns one page of the transfer items in a batch.
func (s *BatchTransfersService) Items(ctx context.Context, batchTransferID string, params *ListParams) (*Page[BatchTransferItem], error) {
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/items"
	return listPage[BatchTransferItem](ctx, s.client, path, params)
}

// AllItems iterates every item in a batch across every page.
func (s *BatchTransfersService) AllItems(ctx context.Context, batchTransferID string, params *ListParams) iter.Seq2[BatchTransferItem, error] {
	page, err := s.Items(ctx, batchTransferID, params)
	return iterPages(ctx, page, err)
}

// Quote locks FX rates for the batch ahead of submission.
func (s *BatchTransfersService) Quote(ctx context.Context, batchTransferID string, params *BatchTransferQuoteParams) (*BatchTransfer, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	batch := &BatchTransfer{}
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/quote"
	if err := s.client.post(ctx, path, body, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// Submit submits the batch for processing.
func (s *BatchTransfersService) Submit(ctx context.Context, batchTransferID string) (*BatchTransfer, error) {
	batch := &BatchTransfer{}
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/submit"
	if err := s.client.post(ctx, path, nil, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// Delete deletes a batch that has not been submitted.
func (s *BatchTransfersService) Delete(ctx context.Context, batchTransferID string) (*BatchTransfer, error) {
	batch := &BatchTransfer{}
	path := batchTransfersBasePath + "/" + pathEscape(batchTransferID) + "/delete"
	if err := s.client.post(ctx, path, nil, batch); err != nil {
		return nil, err
	}
	return batch, nil
}
