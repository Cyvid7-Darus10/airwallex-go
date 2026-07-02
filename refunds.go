package airwallex

import (
	"context"
	"iter"
)

const refundsBasePath = "/api/v1/pa/refunds"

// Refund is a full or partial refund of a payment (/api/v1/pa/refunds).
type Refund struct {
	APIResource
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`

	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Reason   string  `json:"reason"`

	PaymentIntentID         string `json:"payment_intent_id"`
	PaymentAttemptID        string `json:"payment_attempt_id"`
	AcquirerReferenceNumber string `json:"acquirer_reference_number"`

	FailureDetails map[string]any `json:"failure_details"`
	Metadata       map[string]any `json:"metadata"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// RefundCreateParams are the parameters for RefundsService.Create.
type RefundCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID        string         `json:"request_id,omitempty"`
	PaymentIntentID  string         `json:"payment_intent_id,omitempty"`
	PaymentAttemptID string         `json:"payment_attempt_id,omitempty"`
	Amount           float64        `json:"amount,omitempty"`
	Reason           string         `json:"reason,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// RefundListParams filter RefundsService.List.
type RefundListParams struct {
	ListParams
	Status           string `json:"status,omitempty"`
	Currency         string `json:"currency,omitempty"`
	PaymentIntentID  string `json:"payment_intent_id,omitempty"`
	PaymentAttemptID string `json:"payment_attempt_id,omitempty"`
	FromCreatedAt    string `json:"from_created_at,omitempty"`
	ToCreatedAt      string `json:"to_created_at,omitempty"`
}

// RefundsService refunds collected payments.
type RefundsService struct{ client *Client }

// Create creates a refund. A request_id is generated automatically when
// params.RequestID is empty, making the call idempotent — a retry never
// refunds twice.
func (s *RefundsService) Create(ctx context.Context, params *RefundCreateParams) (*Refund, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	refund := &Refund{}
	if err := s.client.post(ctx, refundsBasePath+"/create", body, refund); err != nil {
		return nil, err
	}
	return refund, nil
}

// Retrieve fetches a single refund by id.
func (s *RefundsService) Retrieve(ctx context.Context, refundID string) (*Refund, error) {
	refund := &Refund{}
	if err := s.client.get(ctx, refundsBasePath+"/"+pathEscape(refundID), nil, refund); err != nil {
		return nil, err
	}
	return refund, nil
}

// List returns one page of refunds, filtered by params (may be nil).
func (s *RefundsService) List(ctx context.Context, params *RefundListParams) (*Page[Refund], error) {
	return listPage[Refund](ctx, s.client, refundsBasePath, params)
}

// All iterates every refund across every page, fetching lazily.
func (s *RefundsService) All(ctx context.Context, params *RefundListParams) iter.Seq2[Refund, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
