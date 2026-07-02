package airwallex

import (
	"context"
	"iter"
)

const customersBasePath = "/api/v1/pa/customers"

// Customer is a shopper whose payment details can be saved
// (/api/v1/pa/customers).
type Customer struct {
	APIResource
	ID                 string `json:"id"`
	RequestID          string `json:"request_id"`
	MerchantCustomerID string `json:"merchant_customer_id"`

	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	BusinessName string         `json:"business_name"`
	Email        string         `json:"email"`
	PhoneNumber  string         `json:"phone_number"`
	Address      map[string]any `json:"address"`

	ClientSecret string         `json:"client_secret"`
	Metadata     map[string]any `json:"metadata"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CustomerClientSecret is a short-lived client secret for browser and
// mobile SDK flows.
type CustomerClientSecret struct {
	APIResource
	ClientSecret string `json:"client_secret"`
	ExpiredTime  string `json:"expired_time"`
}

// CustomerCreateParams are the parameters for CustomersService.Create and
// Update.
type CustomerCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty
	// (Create only — Update sends the params as-is).
	RequestID          string         `json:"request_id,omitempty"`
	MerchantCustomerID string         `json:"merchant_customer_id,omitempty"`
	FirstName          string         `json:"first_name,omitempty"`
	LastName           string         `json:"last_name,omitempty"`
	BusinessName       string         `json:"business_name,omitempty"`
	Email              string         `json:"email,omitempty"`
	PhoneNumber        string         `json:"phone_number,omitempty"`
	Address            map[string]any `json:"address,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

// CustomerListParams filter CustomersService.List.
type CustomerListParams struct {
	ListParams
	MerchantCustomerID string `json:"merchant_customer_id,omitempty"`
	FromCreatedAt      string `json:"from_created_at,omitempty"`
	ToCreatedAt        string `json:"to_created_at,omitempty"`
}

// CustomersService manages payment-acceptance shoppers.
type CustomersService struct{ client *Client }

// Create saves a new customer. A request_id is generated automatically
// when params.RequestID is empty, making the call idempotent.
func (s *CustomersService) Create(ctx context.Context, params *CustomerCreateParams) (*Customer, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	customer := &Customer{}
	if err := s.client.post(ctx, customersBasePath+"/create", body, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

// Retrieve fetches a single customer by id.
func (s *CustomersService) Retrieve(ctx context.Context, customerID string) (*Customer, error) {
	customer := &Customer{}
	if err := s.client.get(ctx, customersBasePath+"/"+pathEscape(customerID), nil, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

// Update changes a customer's details.
func (s *CustomersService) Update(ctx context.Context, customerID string, params *CustomerCreateParams) (*Customer, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	customer := &Customer{}
	path := customersBasePath + "/" + pathEscape(customerID) + "/update"
	if err := s.client.post(ctx, path, body, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

// List returns one page of customers, filtered by params (may be nil).
func (s *CustomersService) List(ctx context.Context, params *CustomerListParams) (*Page[Customer], error) {
	return listPage[Customer](ctx, s.client, customersBasePath, params)
}

// All iterates every customer across every page, fetching lazily.
func (s *CustomersService) All(ctx context.Context, params *CustomerListParams) iter.Seq2[Customer, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// GenerateClientSecret creates a short-lived client secret for use in
// browser and mobile SDK flows.
func (s *CustomersService) GenerateClientSecret(ctx context.Context, customerID string) (*CustomerClientSecret, error) {
	secret := &CustomerClientSecret{}
	path := customersBasePath + "/" + pathEscape(customerID) + "/generate_client_secret"
	if err := s.client.get(ctx, path, nil, secret); err != nil {
		return nil, err
	}
	return secret, nil
}
