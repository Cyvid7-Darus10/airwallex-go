package airwallex

import (
	"context"
	"encoding/json"
	"iter"
)

const payersBasePath = "/api/v1/payers"

// PayerDetails describe who money is sent on behalf of.
type PayerDetails struct {
	EntityType     string         `json:"entity_type,omitempty"`
	CompanyName    string         `json:"company_name,omitempty"`
	FirstName      string         `json:"first_name,omitempty"`
	LastName       string         `json:"last_name,omitempty"`
	DateOfBirth    string         `json:"date_of_birth,omitempty"`
	Address        map[string]any `json:"address,omitempty"`
	AdditionalInfo map[string]any `json:"additional_info,omitempty"`
}

// Payer is a saved payer (/api/v1/payers).
type Payer struct {
	APIResource
	PayerID  string        `json:"payer_id"`
	Nickname string        `json:"nickname"`
	Payer    *PayerDetails `json:"payer"`
}

// PayerCreateParams are the parameters for PayersService.Create and Update.
// Note: this endpoint has no request_id — creating twice creates two payers.
type PayerCreateParams struct {
	Params
	Payer    *PayerDetails `json:"payer,omitempty"`
	Nickname string        `json:"nickname,omitempty"`
}

// PayerListParams filter PayersService.List.
type PayerListParams struct {
	ListParams
	EntityType string `json:"entity_type,omitempty"`
	Name       string `json:"name,omitempty"`
	NickName   string `json:"nick_name,omitempty"`
	FromDate   string `json:"from_date,omitempty"`
	ToDate     string `json:"to_date,omitempty"`
}

// PayersService manages the payers money is sent on behalf of.
type PayersService struct{ client *Client }

// Create saves a new payer.
func (s *PayersService) Create(ctx context.Context, params *PayerCreateParams) (*Payer, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	payer := &Payer{}
	if err := s.client.post(ctx, payersBasePath+"/create", body, payer); err != nil {
		return nil, err
	}
	return payer, nil
}

// Retrieve fetches a single payer by id.
func (s *PayersService) Retrieve(ctx context.Context, payerID string) (*Payer, error) {
	payer := &Payer{}
	if err := s.client.get(ctx, payersBasePath+"/"+pathEscape(payerID), nil, payer); err != nil {
		return nil, err
	}
	return payer, nil
}

// Update replaces a payer's details.
func (s *PayersService) Update(ctx context.Context, payerID string, params *PayerCreateParams) (*Payer, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	payer := &Payer{}
	if err := s.client.post(ctx, payersBasePath+"/update/"+pathEscape(payerID), body, payer); err != nil {
		return nil, err
	}
	return payer, nil
}

// Delete removes a saved payer.
func (s *PayersService) Delete(ctx context.Context, payerID string) error {
	return s.client.post(ctx, payersBasePath+"/delete/"+pathEscape(payerID), nil, nil)
}

// List returns one page of payers, filtered by params (may be nil).
func (s *PayersService) List(ctx context.Context, params *PayerListParams) (*Page[Payer], error) {
	return listPage[Payer](ctx, s.client, payersBasePath, params)
}

// All iterates every payer across every page, fetching lazily.
func (s *PayersService) All(ctx context.Context, params *PayerListParams) iter.Seq2[Payer, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Validate validates a payer payload without saving it, returning the raw
// validation result from Airwallex.
func (s *PayersService) Validate(ctx context.Context, params *PayerCreateParams) (json.RawMessage, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := s.client.post(ctx, payersBasePath+"/validate", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
