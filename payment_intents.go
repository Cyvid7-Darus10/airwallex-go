package airwallex

import (
	"context"
	"iter"
)

const paymentIntentsBasePath = "/api/v1/pa/payment_intents"

// PaymentIntent is a payment collected from a shopper
// (/api/v1/pa/payment_intents).
type PaymentIntent struct {
	APIResource
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`

	Amount         float64 `json:"amount"`
	CapturedAmount float64 `json:"captured_amount"`
	Currency       string  `json:"currency"`

	MerchantOrderID    string `json:"merchant_order_id"`
	InvoiceID          string `json:"invoice_id"`
	PaymentLinkID      string `json:"payment_link_id"`
	ConnectedAccountID string `json:"connected_account_id"`
	ConversionQuoteID  string `json:"conversion_quote_id"`
	Descriptor         string `json:"descriptor"`
	ReturnURL          string `json:"return_url"`
	ClientSecret       string `json:"client_secret"`
	TriggeredBy        string `json:"triggered_by"`

	CustomerID           string         `json:"customer_id"`
	Customer             map[string]any `json:"customer"`
	PaymentConsentID     string         `json:"payment_consent_id"`
	PaymentConsent       map[string]any `json:"payment_consent"`
	PaymentMethodOptions map[string]any `json:"payment_method_options"`
	LatestPaymentAttempt map[string]any `json:"latest_payment_attempt"`
	NextAction           map[string]any `json:"next_action"`

	Order              map[string]any   `json:"order"`
	AdditionalInfo     map[string]any   `json:"additional_info"`
	FundsSplitData     []map[string]any `json:"funds_split_data"`
	RiskControlOptions map[string]any   `json:"risk_control_options"`
	Metadata           map[string]any   `json:"metadata"`

	CancellationReason string `json:"cancellation_reason"`
	CancelledAt        string `json:"cancelled_at"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// PaymentIntentCreateParams are the parameters for
// PaymentIntentsService.Create.
type PaymentIntentCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID       string         `json:"request_id,omitempty"`
	Amount          float64        `json:"amount,omitempty"`
	Currency        string         `json:"currency,omitempty"`
	MerchantOrderID string         `json:"merchant_order_id,omitempty"`
	CustomerID      string         `json:"customer_id,omitempty"`
	Descriptor      string         `json:"descriptor,omitempty"`
	ReturnURL       string         `json:"return_url,omitempty"`
	Order           map[string]any `json:"order,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// PaymentIntentActionParams carry the payload for confirm / continue /
// capture / cancel actions on a payment intent (e.g. payment_method,
// payment_consent_reference, amount). Use ExtraParams for fields this SDK
// has no typed field for.
type PaymentIntentActionParams struct {
	Params
	// RequestID makes the action idempotent; auto-generated when empty.
	RequestID          string         `json:"request_id,omitempty"`
	Amount             float64        `json:"amount,omitempty"`
	PaymentMethod      map[string]any `json:"payment_method,omitempty"`
	PaymentConsentRef  map[string]any `json:"payment_consent_reference,omitempty"`
	PaymentMethodOpts  map[string]any `json:"payment_method_options,omitempty"`
	CancellationReason string         `json:"cancellation_reason,omitempty"`
	Type               string         `json:"type,omitempty"`
	ThreeDS            map[string]any `json:"three_ds,omitempty"`
	DeviceData         map[string]any `json:"device_data,omitempty"`
}

// PaymentIntentListParams filter PaymentIntentsService.List.
type PaymentIntentListParams struct {
	ListParams
	Status             string `json:"status,omitempty"`
	Currency           string `json:"currency,omitempty"`
	MerchantOrderID    string `json:"merchant_order_id,omitempty"`
	PaymentConsentID   string `json:"payment_consent_id,omitempty"`
	ConnectedAccountID string `json:"connected_account_id,omitempty"`
	FromCreatedAt      string `json:"from_created_at,omitempty"`
	ToCreatedAt        string `json:"to_created_at,omitempty"`
}

// PaymentIntentsService collects payments from shoppers.
type PaymentIntentsService struct{ client *Client }

// Create creates a payment intent. A request_id is generated automatically
// when params.RequestID is empty, making the call idempotent.
func (s *PaymentIntentsService) Create(ctx context.Context, params *PaymentIntentCreateParams) (*PaymentIntent, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	intent := &PaymentIntent{}
	if err := s.client.post(ctx, paymentIntentsBasePath+"/create", body, intent); err != nil {
		return nil, err
	}
	return intent, nil
}

// Retrieve fetches a single payment intent by id.
func (s *PaymentIntentsService) Retrieve(ctx context.Context, paymentIntentID string) (*PaymentIntent, error) {
	intent := &PaymentIntent{}
	if err := s.client.get(ctx, paymentIntentsBasePath+"/"+pathEscape(paymentIntentID), nil, intent); err != nil {
		return nil, err
	}
	return intent, nil
}

// List returns one page of payment intents, filtered by params (may be
// nil).
func (s *PaymentIntentsService) List(ctx context.Context, params *PaymentIntentListParams) (*Page[PaymentIntent], error) {
	return listPage[PaymentIntent](ctx, s.client, paymentIntentsBasePath, params)
}

// All iterates every payment intent across every page, fetching lazily.
func (s *PaymentIntentsService) All(ctx context.Context, params *PaymentIntentListParams) iter.Seq2[PaymentIntent, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Confirm confirms a payment intent with a payment method.
func (s *PaymentIntentsService) Confirm(ctx context.Context, paymentIntentID string, params *PaymentIntentActionParams) (*PaymentIntent, error) {
	return s.action(ctx, paymentIntentID, "confirm", params)
}

// ConfirmContinue continues a confirmation that requires further steps
// (e.g. 3-D Secure).
func (s *PaymentIntentsService) ConfirmContinue(ctx context.Context, paymentIntentID string, params *PaymentIntentActionParams) (*PaymentIntent, error) {
	return s.action(ctx, paymentIntentID, "confirm_continue", params)
}

// Capture captures a previously authorized payment intent.
func (s *PaymentIntentsService) Capture(ctx context.Context, paymentIntentID string, params *PaymentIntentActionParams) (*PaymentIntent, error) {
	return s.action(ctx, paymentIntentID, "capture", params)
}

// Cancel cancels a payment intent.
func (s *PaymentIntentsService) Cancel(ctx context.Context, paymentIntentID string, params *PaymentIntentActionParams) (*PaymentIntent, error) {
	return s.action(ctx, paymentIntentID, "cancel", params)
}

func (s *PaymentIntentsService) action(ctx context.Context, paymentIntentID, action string, params *PaymentIntentActionParams) (*PaymentIntent, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	intent := &PaymentIntent{}
	path := paymentIntentsBasePath + "/" + pathEscape(paymentIntentID) + "/" + action
	if err := s.client.post(ctx, path, body, intent); err != nil {
		return nil, err
	}
	return intent, nil
}
