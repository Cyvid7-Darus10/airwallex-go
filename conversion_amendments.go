package airwallex

import (
	"context"
	"iter"
)

const conversionAmendmentsBasePath = "/api/v1/conversion_amendments"

// AmendmentCharge is one charge or credit resulting from an amendment.
type AmendmentCharge struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Type         string  `json:"type"`
	CurrencyPair string  `json:"currency_pair"`
	AwxRate      float64 `json:"awx_rate"`
	ClientRate   float64 `json:"client_rate"`
}

// ConversionAmendment amends or cancels an existing conversion
// (/api/v1/conversion_amendments).
type ConversionAmendment struct {
	APIResource
	AmendmentID      string `json:"amendment_id"`
	RequestID        string `json:"request_id"`
	ShortReferenceID string `json:"short_reference_id"`
	ConversionID     string `json:"conversion_id"`
	Type             string `json:"type"`

	Charges  []AmendmentCharge `json:"charges"`
	Metadata map[string]any    `json:"metadata"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ConversionAmendmentQuote previews the charges of an amendment before
// committing to it.
type ConversionAmendmentQuote struct {
	APIResource
	RequestID        string            `json:"request_id"`
	ShortReferenceID string            `json:"short_reference_id"`
	ConversionID     string            `json:"conversion_id"`
	Type             string            `json:"type"`
	Charges          []AmendmentCharge `json:"charges"`
	Metadata         map[string]any    `json:"metadata"`
}

// ConversionAmendmentCreateParams are the parameters for
// ConversionAmendmentsService.Create and Quote.
type ConversionAmendmentCreateParams struct {
	Params
	// RequestID makes the call idempotent; auto-generated when empty.
	RequestID    string `json:"request_id,omitempty"`
	ConversionID string `json:"conversion_id,omitempty"`
	// Type is the amendment kind, e.g. "CANCELLATION".
	Type     string         `json:"type,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ConversionAmendmentListParams filter ConversionAmendmentsService.List.
// ConversionID is required by the API.
type ConversionAmendmentListParams struct {
	ListParams
	ConversionID string `json:"conversion_id,omitempty"`
}

// ConversionAmendmentsService amends or cancels existing conversions.
type ConversionAmendmentsService struct{ client *Client }

// Create executes an amendment (e.g. cancel a conversion). A request_id is
// generated automatically when params.RequestID is empty.
func (s *ConversionAmendmentsService) Create(ctx context.Context, params *ConversionAmendmentCreateParams) (*ConversionAmendment, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	amendment := &ConversionAmendment{}
	if err := s.client.post(ctx, conversionAmendmentsBasePath+"/create", body, amendment); err != nil {
		return nil, err
	}
	return amendment, nil
}

// Quote previews the charges an amendment would incur, without executing
// it. A request_id is generated automatically when params.RequestID is
// empty.
func (s *ConversionAmendmentsService) Quote(ctx context.Context, params *ConversionAmendmentCreateParams) (*ConversionAmendmentQuote, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	quote := &ConversionAmendmentQuote{}
	if err := s.client.post(ctx, conversionAmendmentsBasePath+"/quote", body, quote); err != nil {
		return nil, err
	}
	return quote, nil
}

// Retrieve fetches a single amendment by id.
func (s *ConversionAmendmentsService) Retrieve(ctx context.Context, conversionAmendmentID string) (*ConversionAmendment, error) {
	amendment := &ConversionAmendment{}
	if err := s.client.get(ctx, conversionAmendmentsBasePath+"/"+pathEscape(conversionAmendmentID), nil, amendment); err != nil {
		return nil, err
	}
	return amendment, nil
}

// List returns one page of amendments for a conversion.
func (s *ConversionAmendmentsService) List(ctx context.Context, params *ConversionAmendmentListParams) (*Page[ConversionAmendment], error) {
	return listPage[ConversionAmendment](ctx, s.client, conversionAmendmentsBasePath, params)
}

// All iterates every amendment across every page, fetching lazily.
func (s *ConversionAmendmentsService) All(ctx context.Context, params *ConversionAmendmentListParams) iter.Seq2[ConversionAmendment, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
