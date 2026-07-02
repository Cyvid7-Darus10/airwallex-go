package airwallex

import "context"

const fxQuotesBasePath = "/api/v1/fx/quotes"

// FxQuote is a lockable FX quote (/api/v1/fx/quotes). Unlike RateQuote,
// the rate is held for the validity window and can be used to create a
// conversion at that rate.
type FxQuote struct {
	APIResource
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`

	CurrencyPair  string  `json:"currency_pair"`
	BuyAmount     float64 `json:"buy_amount"`
	BuyCurrency   string  `json:"buy_currency"`
	SellAmount    float64 `json:"sell_amount"`
	SellCurrency  string  `json:"sell_currency"`
	DealtCurrency string  `json:"dealt_currency"`

	AwxRate     float64          `json:"awx_rate"`
	ClientRate  float64          `json:"client_rate"`
	MidRate     float64          `json:"mid_rate"`
	RateDetails []map[string]any `json:"rate_details"`

	Validity       string `json:"validity"`
	ConversionDate string `json:"conversion_date"`
	ExpiresAt      string `json:"expires_at"`
	CreatedAt      string `json:"created_at"`
}

// FxQuoteCreateParams are the parameters for FxQuotesService.Create.
type FxQuoteCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID    string  `json:"request_id,omitempty"`
	BuyCurrency  string  `json:"buy_currency,omitempty"`
	BuyAmount    float64 `json:"buy_amount,omitempty"`
	SellCurrency string  `json:"sell_currency,omitempty"`
	SellAmount   float64 `json:"sell_amount,omitempty"`
	// Validity is how long the quote is locked, e.g. "HR_1".
	Validity       string `json:"validity,omitempty"`
	ConversionDate string `json:"conversion_date,omitempty"`
}

// FxQuotesService creates lockable FX quotes.
type FxQuotesService struct{ client *Client }

// Create locks an FX quote. A request_id is generated automatically when
// params.RequestID is empty, making the call idempotent.
func (s *FxQuotesService) Create(ctx context.Context, params *FxQuoteCreateParams) (*FxQuote, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	quote := &FxQuote{}
	if err := s.client.post(ctx, fxQuotesBasePath+"/create", body, quote); err != nil {
		return nil, err
	}
	return quote, nil
}

// Retrieve fetches a single FX quote by id.
func (s *FxQuotesService) Retrieve(ctx context.Context, quoteID string) (*FxQuote, error) {
	quote := &FxQuote{}
	if err := s.client.get(ctx, fxQuotesBasePath+"/"+pathEscape(quoteID), nil, quote); err != nil {
		return nil, err
	}
	return quote, nil
}
