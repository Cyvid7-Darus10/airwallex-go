package airwallex

import (
	"context"
	"iter"
)

const (
	conversionsBasePath  = "/api/v1/conversions"
	ratesCurrentPath     = "/api/v1/fx/rates/current"
	conversionAmendments = "/api/v1/conversion_amendments"
)

// Conversion is a booked FX conversion (/api/v1/conversions).
type Conversion struct {
	APIResource
	ConversionID     string `json:"conversion_id"`
	RequestID        string `json:"request_id"`
	ShortReferenceID string `json:"short_reference_id"`
	Status           string `json:"status"`

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

	QuoteID              string `json:"quote_id"`
	ConversionDate       string `json:"conversion_date"`
	SettlementCutoffTime string `json:"settlement_cutoff_time"`
	Reason               string `json:"reason"`

	CreatedAt     string `json:"created_at"`
	LastUpdatedAt string `json:"last_updated_at"`
}

// RateQuote is an indicative FX rate (GET /api/v1/fx/rates/current). No
// funds move; use FxQuotes to lock a rate.
//
// Current API versions return Rate (with per-level detail in RateDetails);
// older versions return ClientRate/MidRate. Both are typed here.
type RateQuote struct {
	APIResource
	CurrencyPair         string           `json:"currency_pair"`
	Rate                 float64          `json:"rate"`
	BuyCurrency          string           `json:"buy_currency"`
	BuyAmount            float64          `json:"buy_amount"`
	SellCurrency         string           `json:"sell_currency"`
	SellAmount           float64          `json:"sell_amount"`
	ConversionDate       string           `json:"conversion_date"`
	CreatedAt            string           `json:"created_at"`
	ClientRate           float64          `json:"client_rate"`
	MidRate              float64          `json:"mid_rate"`
	DealtCurrency        string           `json:"dealt_currency"`
	ClientBuyAmount      float64          `json:"client_buy_amount"`
	ClientBuyCurrency    string           `json:"client_buy_currency"`
	ClientSellAmount     float64          `json:"client_sell_amount"`
	ClientSellCurrency   string           `json:"client_sell_currency"`
	SettlementCutoffTime string           `json:"settlement_cutoff_time"`
	SettlementDate       string           `json:"settlement_date"`
	RateDetails          []map[string]any `json:"rate_details"`
}

// ConversionCreateParams are the parameters for ConversionsService.Create.
type ConversionCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID    string  `json:"request_id,omitempty"`
	BuyCurrency  string  `json:"buy_currency,omitempty"`
	BuyAmount    float64 `json:"buy_amount,omitempty"`
	SellCurrency string  `json:"sell_currency,omitempty"`
	SellAmount   float64 `json:"sell_amount,omitempty"`
	QuoteID      string  `json:"quote_id,omitempty"`
	// TermAgreement must be true to accept the conversion terms.
	TermAgreement  bool   `json:"term_agreement,omitempty"`
	ConversionDate string `json:"conversion_date,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// ConversionListParams filter ConversionsService.List.
type ConversionListParams struct {
	ListParams
	Status        string `json:"status,omitempty"`
	BuyCurrency   string `json:"buy_currency,omitempty"`
	SellCurrency  string `json:"sell_currency,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
}

// RateCurrentParams are the parameters for RatesService.Current. Specify at
// most one of BuyAmount / SellAmount; Airwallex defaults to a notional
// amount of 10,000 when neither is given.
type RateCurrentParams struct {
	BuyCurrency    string  `json:"buy_currency,omitempty"`
	SellCurrency   string  `json:"sell_currency,omitempty"`
	BuyAmount      float64 `json:"buy_amount,omitempty"`
	SellAmount     float64 `json:"sell_amount,omitempty"`
	ConversionDate string  `json:"conversion_date,omitempty"`
}

// ConversionsService books FX conversions between wallet currencies.
type ConversionsService struct{ client *Client }

// Create books a conversion. A request_id is generated automatically when
// params.RequestID is empty, making the call idempotent.
func (s *ConversionsService) Create(ctx context.Context, params *ConversionCreateParams) (*Conversion, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	conversion := &Conversion{}
	if err := s.client.post(ctx, conversionsBasePath+"/create", body, conversion); err != nil {
		return nil, err
	}
	return conversion, nil
}

// Retrieve fetches a single conversion by id.
func (s *ConversionsService) Retrieve(ctx context.Context, conversionID string) (*Conversion, error) {
	conversion := &Conversion{}
	if err := s.client.get(ctx, conversionsBasePath+"/"+pathEscape(conversionID), nil, conversion); err != nil {
		return nil, err
	}
	return conversion, nil
}

// List returns one page of conversions, filtered by params (may be nil).
func (s *ConversionsService) List(ctx context.Context, params *ConversionListParams) (*Page[Conversion], error) {
	return listPage[Conversion](ctx, s.client, conversionsBasePath, params)
}

// All iterates every conversion across every page, fetching lazily.
func (s *ConversionsService) All(ctx context.Context, params *ConversionListParams) iter.Seq2[Conversion, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// RatesService fetches indicative FX rates.
type RatesService struct{ client *Client }

// Current gets the current indicative FX rate for a currency pair. No
// funds move.
func (s *RatesService) Current(ctx context.Context, params *RateCurrentParams) (*RateQuote, error) {
	query, err := encodeQuery(params)
	if err != nil {
		return nil, err
	}
	quote := &RateQuote{}
	if err := s.client.get(ctx, ratesCurrentPath, query, quote); err != nil {
		return nil, err
	}
	return quote, nil
}
