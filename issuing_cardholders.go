package airwallex

import (
	"context"
	"iter"
)

const issuingCardholdersBasePath = "/api/v1/issuing/cardholders"

// Cardholder is a person cards can be issued to
// (/api/v1/issuing/cardholders).
type Cardholder struct {
	APIResource
	CardholderID string `json:"cardholder_id"`
	Email        string `json:"email"`
	MobileNumber string `json:"mobile_number"`
	Status       string `json:"status"`

	Individual    map[string]any `json:"individual"`
	Address       map[string]any `json:"address"`
	PostalAddress map[string]any `json:"postal_address"`
}

// CardholderCreateParams are the parameters for
// IssuingCardholdersService.Create and Update. Note: this endpoint has no
// request_id — creating twice creates two cardholders.
type CardholderCreateParams struct {
	Params
	Email         string         `json:"email,omitempty"`
	MobileNumber  string         `json:"mobile_number,omitempty"`
	Individual    map[string]any `json:"individual,omitempty"`
	Address       map[string]any `json:"address,omitempty"`
	PostalAddress map[string]any `json:"postal_address,omitempty"`
	// Type is the cardholder kind, e.g. "INDIVIDUAL" or "DELEGATE".
	Type string `json:"type,omitempty"`
}

// CardholderListParams filter IssuingCardholdersService.List.
type CardholderListParams struct {
	ListParams
	CardholderStatus string `json:"cardholder_status,omitempty"`
	Email            string `json:"email,omitempty"`
}

// IssuingCardholdersService manages people cards can be issued to.
type IssuingCardholdersService struct{ client *Client }

// Create registers a new cardholder.
func (s *IssuingCardholdersService) Create(ctx context.Context, params *CardholderCreateParams) (*Cardholder, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	cardholder := &Cardholder{}
	if err := s.client.post(ctx, issuingCardholdersBasePath+"/create", body, cardholder); err != nil {
		return nil, err
	}
	return cardholder, nil
}

// Retrieve fetches a single cardholder by id.
func (s *IssuingCardholdersService) Retrieve(ctx context.Context, cardholderID string) (*Cardholder, error) {
	cardholder := &Cardholder{}
	if err := s.client.get(ctx, issuingCardholdersBasePath+"/"+pathEscape(cardholderID), nil, cardholder); err != nil {
		return nil, err
	}
	return cardholder, nil
}

// Update changes a cardholder's details.
func (s *IssuingCardholdersService) Update(ctx context.Context, cardholderID string, params *CardholderCreateParams) (*Cardholder, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	cardholder := &Cardholder{}
	path := issuingCardholdersBasePath + "/" + pathEscape(cardholderID) + "/update"
	if err := s.client.post(ctx, path, body, cardholder); err != nil {
		return nil, err
	}
	return cardholder, nil
}

// Delete removes a cardholder who has no active cards.
func (s *IssuingCardholdersService) Delete(ctx context.Context, cardholderID string) (*Cardholder, error) {
	cardholder := &Cardholder{}
	path := issuingCardholdersBasePath + "/" + pathEscape(cardholderID) + "/delete"
	if err := s.client.post(ctx, path, nil, cardholder); err != nil {
		return nil, err
	}
	return cardholder, nil
}

// List returns one page of cardholders, filtered by params (may be nil).
func (s *IssuingCardholdersService) List(ctx context.Context, params *CardholderListParams) (*Page[Cardholder], error) {
	return listPage[Cardholder](ctx, s.client, issuingCardholdersBasePath, params)
}

// All iterates every cardholder across every page, fetching lazily.
func (s *IssuingCardholdersService) All(ctx context.Context, params *CardholderListParams) iter.Seq2[Cardholder, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
