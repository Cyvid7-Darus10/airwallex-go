package airwallex

import "context"

const accountBasePath = "/api/v1/account"

// Account is your own Airwallex account (GET /api/v1/account).
type Account struct {
	APIResource
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Nickname   string `json:"nickname"`
	Status     string `json:"status"`
	ViewType   string `json:"view_type"`

	AccountDetails    map[string]any   `json:"account_details"`
	PrimaryContact    map[string]any   `json:"primary_contact"`
	ReactivateDetails map[string]any   `json:"reactivate_details"`
	SuspendDetails    []map[string]any `json:"suspend_details"`
	Metadata          map[string]any   `json:"metadata"`

	CreatedAt string `json:"created_at"`
}

// AccountsService retrieves details of your own Airwallex account.
type AccountsService struct{ client *Client }

// Retrieve fetches the account the credentials belong to.
func (s *AccountsService) Retrieve(ctx context.Context) (*Account, error) {
	account := &Account{}
	if err := s.client.get(ctx, accountBasePath, nil, account); err != nil {
		return nil, err
	}
	return account, nil
}
