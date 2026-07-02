package airwallex

import (
	"context"
	"iter"
)

const webhookEndpointsBasePath = "/api/v1/webhooks"

// WebhookEndpoint is a webhook subscription (/api/v1/webhooks). Secret is
// only returned on create — store it to verify signatures with the
// webhooks package.
type WebhookEndpoint struct {
	APIResource
	ID        string   `json:"id"`
	RequestID string   `json:"request_id"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret"`
	Version   string   `json:"version"`
	Events    []string `json:"events"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// WebhookEndpointCreateParams are the parameters for
// WebhookEndpointsService.Create and Update.
type WebhookEndpointCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty
	// (Create only — Update sends the params as-is).
	RequestID string `json:"request_id,omitempty"`
	// URL is where notifications are delivered.
	URL string `json:"url,omitempty"`
	// Events are the event names to subscribe to (e.g. "transfer.settled").
	Events []string `json:"events,omitempty"`
}

// WebhookEndpointsService manages webhook subscriptions.
type WebhookEndpointsService struct{ client *Client }

// Create registers a webhook endpoint. A request_id is generated
// automatically when params.RequestID is empty.
func (s *WebhookEndpointsService) Create(ctx context.Context, params *WebhookEndpointCreateParams) (*WebhookEndpoint, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	endpoint := &WebhookEndpoint{}
	if err := s.client.post(ctx, webhookEndpointsBasePath+"/create", body, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// Retrieve fetches a single webhook endpoint by id.
func (s *WebhookEndpointsService) Retrieve(ctx context.Context, webhookID string) (*WebhookEndpoint, error) {
	endpoint := &WebhookEndpoint{}
	if err := s.client.get(ctx, webhookEndpointsBasePath+"/"+pathEscape(webhookID), nil, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// Update changes a webhook endpoint's URL or subscribed events.
func (s *WebhookEndpointsService) Update(ctx context.Context, webhookID string, params *WebhookEndpointCreateParams) (*WebhookEndpoint, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	endpoint := &WebhookEndpoint{}
	path := webhookEndpointsBasePath + "/" + pathEscape(webhookID) + "/update"
	if err := s.client.post(ctx, path, body, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// Delete removes a webhook endpoint.
func (s *WebhookEndpointsService) Delete(ctx context.Context, webhookID string) error {
	return s.client.post(ctx, webhookEndpointsBasePath+"/"+pathEscape(webhookID)+"/delete", nil, nil)
}

// List returns one page of webhook endpoints. params may be nil.
func (s *WebhookEndpointsService) List(ctx context.Context, params *ListParams) (*Page[WebhookEndpoint], error) {
	return listPage[WebhookEndpoint](ctx, s.client, webhookEndpointsBasePath, params)
}

// All iterates every webhook endpoint across every page, fetching lazily.
func (s *WebhookEndpointsService) All(ctx context.Context, params *ListParams) iter.Seq2[WebhookEndpoint, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
