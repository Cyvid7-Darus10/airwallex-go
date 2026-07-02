package airwallex

import (
	"context"
	"iter"
)

const issuingCardsBasePath = "/api/v1/issuing/cards"

// Card is an issued card (/api/v1/issuing/cards). PCI-scoped endpoints
// (/details, /provision_digital_token) are deliberately not wrapped.
type Card struct {
	APIResource
	CardID       string `json:"card_id"`
	RequestID    string `json:"request_id"`
	CardStatus   string `json:"card_status"`
	CardNumber   string `json:"card_number"`
	CardholderID string `json:"cardholder_id"`

	Brand      string `json:"brand"`
	FormFactor string `json:"form_factor"`
	Type       string `json:"type"`
	IssueTo    string `json:"issue_to"`
	Purpose    string `json:"purpose"`

	NameOnCard      string `json:"name_on_card"`
	NickName        string `json:"nick_name"`
	Note            string `json:"note"`
	ClientData      string `json:"client_data"`
	CreatedBy       string `json:"created_by"`
	ActivateOnIssue bool   `json:"activate_on_issue"`

	AuthorizationControls map[string]any `json:"authorization_controls"`
	PostalAddress         map[string]any `json:"postal_address"`
	PrimaryContactDetails map[string]any `json:"primary_contact_details"`
	DeliveryDetails       map[string]any `json:"delivery_details"`
	Metadata              map[string]any `json:"metadata"`

	CardVersion     int              `json:"card_version"`
	AllCardVersions []map[string]any `json:"all_card_versions"`

	CreatedAt string `json:"created_at"`
}

// CardLimits are a card's spending limits
// (GET /api/v1/issuing/cards/{id}/limits).
type CardLimits struct {
	APIResource
	Currency             string           `json:"currency"`
	Limits               []map[string]any `json:"limits"`
	CashWithdrawalLimits []map[string]any `json:"cash_withdrawal_limits"`
}

// CardCreateParams are the parameters for IssuingCardsService.Create and
// Update.
type CardCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty
	// (Create only — Update sends the params as-is).
	RequestID    string `json:"request_id,omitempty"`
	CardholderID string `json:"cardholder_id,omitempty"`
	FormFactor   string `json:"form_factor,omitempty"`
	IssueTo      string `json:"issue_to,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
	NameOnCard   string `json:"name_on_card,omitempty"`
	NickName     string `json:"nick_name,omitempty"`
	Note         string `json:"note,omitempty"`
	ClientData   string `json:"client_data,omitempty"`

	ActivateOnIssue       bool           `json:"activate_on_issue,omitempty"`
	Program               map[string]any `json:"program,omitempty"`
	AuthorizationControls map[string]any `json:"authorization_controls,omitempty"`
	PostalAddress         map[string]any `json:"postal_address,omitempty"`
	PrimaryContactDetails map[string]any `json:"primary_contact_details,omitempty"`
	DeliveryDetails       map[string]any `json:"delivery_details,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

// CardListParams filter IssuingCardsService.List.
type CardListParams struct {
	ListParams
	CardStatus    string `json:"card_status,omitempty"`
	CardholderID  string `json:"cardholder_id,omitempty"`
	NickName      string `json:"nick_name,omitempty"`
	FromCreatedAt string `json:"from_created_at,omitempty"`
	ToCreatedAt   string `json:"to_created_at,omitempty"`
	FromUpdatedAt string `json:"from_updated_at,omitempty"`
	ToUpdatedAt   string `json:"to_updated_at,omitempty"`
}

// IssuingCardsService manages issued cards.
type IssuingCardsService struct{ client *Client }

// Create issues a card. A request_id is generated automatically when
// params.RequestID is empty, making the call idempotent — a retry never
// issues two cards.
func (s *IssuingCardsService) Create(ctx context.Context, params *CardCreateParams) (*Card, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	card := &Card{}
	if err := s.client.post(ctx, issuingCardsBasePath+"/create", body, card); err != nil {
		return nil, err
	}
	return card, nil
}

// Retrieve fetches a single card by id.
func (s *IssuingCardsService) Retrieve(ctx context.Context, cardID string) (*Card, error) {
	card := &Card{}
	if err := s.client.get(ctx, issuingCardsBasePath+"/"+pathEscape(cardID), nil, card); err != nil {
		return nil, err
	}
	return card, nil
}

// Update changes a card's mutable details.
func (s *IssuingCardsService) Update(ctx context.Context, cardID string, params *CardCreateParams) (*Card, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	card := &Card{}
	path := issuingCardsBasePath + "/" + pathEscape(cardID) + "/update"
	if err := s.client.post(ctx, path, body, card); err != nil {
		return nil, err
	}
	return card, nil
}

// Activate activates a physical card.
func (s *IssuingCardsService) Activate(ctx context.Context, cardID string) error {
	return s.client.post(ctx, issuingCardsBasePath+"/"+pathEscape(cardID)+"/activate", nil, nil)
}

// Limits fetches a card's spending limits.
func (s *IssuingCardsService) Limits(ctx context.Context, cardID string) (*CardLimits, error) {
	limits := &CardLimits{}
	if err := s.client.get(ctx, issuingCardsBasePath+"/"+pathEscape(cardID)+"/limits", nil, limits); err != nil {
		return nil, err
	}
	return limits, nil
}

// List returns one page of cards, filtered by params (may be nil).
func (s *IssuingCardsService) List(ctx context.Context, params *CardListParams) (*Page[Card], error) {
	return listPage[Card](ctx, s.client, issuingCardsBasePath, params)
}

// All iterates every card across every page, fetching lazily.
func (s *IssuingCardsService) All(ctx context.Context, params *CardListParams) iter.Seq2[Card, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
