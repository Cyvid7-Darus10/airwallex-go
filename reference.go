package airwallex

import (
	"context"
	"encoding/json"
	"net/url"
)

const referenceBasePath = "/api/v1/reference"

// ReferenceSettlementAccountsParams filter
// ReferenceService.SettlementAccounts.
type ReferenceSettlementAccountsParams struct {
	CountryCode string `json:"country_code,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

// ReferenceService exposes static reference data.
type ReferenceService struct{ client *Client }

// SupportedCurrencies lists the currencies Airwallex supports, as the raw
// JSON reference payload.
func (s *ReferenceService) SupportedCurrencies(ctx context.Context) (json.RawMessage, error) {
	var result json.RawMessage
	if err := s.client.get(ctx, referenceBasePath+"/supported_currencies", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SettlementAccounts lists settlement accounts available for the given
// corridor, as the raw JSON reference payload. params may be nil.
func (s *ReferenceService) SettlementAccounts(ctx context.Context, params *ReferenceSettlementAccountsParams) (json.RawMessage, error) {
	query, err := encodeQuery(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := s.client.get(ctx, referenceBasePath+"/settlement_accounts", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// InvalidConversionDates lists dates on which the given currency pair
// (e.g. "USDSGD") cannot settle, as the raw JSON reference payload.
func (s *ReferenceService) InvalidConversionDates(ctx context.Context, currencyPair string) (json.RawMessage, error) {
	query := url.Values{"currency_pair": {currencyPair}}
	var result json.RawMessage
	if err := s.client.get(ctx, referenceBasePath+"/invalid_conversion_dates", query, &result); err != nil {
		return nil, err
	}
	return result, nil
}
