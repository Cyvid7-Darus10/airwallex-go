package airwallex

import (
	"context"
	"encoding/json"
	"iter"
)

const beneficiariesBasePath = "/api/v1/beneficiaries"

// BankDetails describe the bank account of a beneficiary.
type BankDetails struct {
	AccountCurrency     string `json:"account_currency,omitempty"`
	AccountName         string `json:"account_name,omitempty"`
	AccountNumber       string `json:"account_number,omitempty"`
	AccountRoutingType1 string `json:"account_routing_type1,omitempty"`
	AccountRoutingVal1  string `json:"account_routing_value1,omitempty"`
	AccountRoutingType2 string `json:"account_routing_type2,omitempty"`
	AccountRoutingVal2  string `json:"account_routing_value2,omitempty"`
	BankCountryCode     string `json:"bank_country_code,omitempty"`
	BankName            string `json:"bank_name,omitempty"`
	BankBranch          string `json:"bank_branch,omitempty"`
	IBAN                string `json:"iban,omitempty"`
	SwiftCode           string `json:"swift_code,omitempty"`
	LocalClearingSystem string `json:"local_clearing_system,omitempty"`
}

// BeneficiaryDetails describe who a payout goes to.
type BeneficiaryDetails struct {
	EntityType     string         `json:"entity_type,omitempty"`
	CompanyName    string         `json:"company_name,omitempty"`
	FirstName      string         `json:"first_name,omitempty"`
	LastName       string         `json:"last_name,omitempty"`
	DateOfBirth    string         `json:"date_of_birth,omitempty"`
	BankDetails    *BankDetails   `json:"bank_details,omitempty"`
	Address        map[string]any `json:"address,omitempty"`
	AdditionalInfo map[string]any `json:"additional_info,omitempty"`
}

// Beneficiary is a saved payout recipient (/api/v1/beneficiaries).
//
// Current API versions return the identifier as id and the methods as
// transfer_methods; older versions use beneficiary_id / payment_methods.
// All are typed. Use the EffectiveID helper to get whichever is set.
type Beneficiary struct {
	APIResource
	ID              string              `json:"id"`
	BeneficiaryID   string              `json:"beneficiary_id"`
	Nickname        string              `json:"nickname"`
	PayerEntityType string              `json:"payer_entity_type"`
	TransferMethods []string            `json:"transfer_methods"`
	PaymentMethods  []string            `json:"payment_methods"`
	Beneficiary     *BeneficiaryDetails `json:"beneficiary"`
}

// EffectiveID returns the beneficiary identifier regardless of which API
// version produced the response.
func (b *Beneficiary) EffectiveID() string {
	if b.ID != "" {
		return b.ID
	}
	return b.BeneficiaryID
}

// BeneficiaryCreateParams are the parameters for
// BeneficiariesService.Create and Update. Note: this endpoint has no
// request_id — creating twice creates two beneficiaries.
type BeneficiaryCreateParams struct {
	Params
	Beneficiary     *BeneficiaryDetails `json:"beneficiary,omitempty"`
	Nickname        string              `json:"nickname,omitempty"`
	PayerEntityType string              `json:"payer_entity_type,omitempty"`
	PaymentMethods  []string            `json:"payment_methods,omitempty"`
	TransferMethods []string            `json:"transfer_methods,omitempty"`
}

// BeneficiaryListParams filter BeneficiariesService.List.
type BeneficiaryListParams struct {
	ListParams
	EntityType        string `json:"entity_type,omitempty"`
	Name              string `json:"name,omitempty"`
	NickName          string `json:"nick_name,omitempty"`
	CompanyName       string `json:"company_name,omitempty"`
	BankAccountNumber string `json:"bank_account_number,omitempty"`
	FromDate          string `json:"from_date,omitempty"`
	ToDate            string `json:"to_date,omitempty"`
}

// BeneficiariesService manages payout recipients.
type BeneficiariesService struct{ client *Client }

// Create saves a new beneficiary.
func (s *BeneficiariesService) Create(ctx context.Context, params *BeneficiaryCreateParams) (*Beneficiary, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	beneficiary := &Beneficiary{}
	if err := s.client.post(ctx, beneficiariesBasePath+"/create", body, beneficiary); err != nil {
		return nil, err
	}
	return beneficiary, nil
}

// Retrieve fetches a single beneficiary by id.
func (s *BeneficiariesService) Retrieve(ctx context.Context, beneficiaryID string) (*Beneficiary, error) {
	beneficiary := &Beneficiary{}
	if err := s.client.get(ctx, beneficiariesBasePath+"/"+pathEscape(beneficiaryID), nil, beneficiary); err != nil {
		return nil, err
	}
	return beneficiary, nil
}

// Update replaces a beneficiary's details.
func (s *BeneficiariesService) Update(ctx context.Context, beneficiaryID string, params *BeneficiaryCreateParams) (*Beneficiary, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	beneficiary := &Beneficiary{}
	path := beneficiariesBasePath + "/update/" + pathEscape(beneficiaryID)
	if err := s.client.post(ctx, path, body, beneficiary); err != nil {
		return nil, err
	}
	return beneficiary, nil
}

// Delete removes a saved beneficiary.
func (s *BeneficiariesService) Delete(ctx context.Context, beneficiaryID string) error {
	return s.client.post(ctx, beneficiariesBasePath+"/delete/"+pathEscape(beneficiaryID), nil, nil)
}

// List returns one page of beneficiaries, filtered by params (may be nil).
func (s *BeneficiariesService) List(ctx context.Context, params *BeneficiaryListParams) (*Page[Beneficiary], error) {
	return listPage[Beneficiary](ctx, s.client, beneficiariesBasePath, params)
}

// All iterates every beneficiary across every page, fetching lazily.
func (s *BeneficiariesService) All(ctx context.Context, params *BeneficiaryListParams) iter.Seq2[Beneficiary, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}

// Validate validates a beneficiary payload without saving it, returning the
// raw validation result from Airwallex.
func (s *BeneficiariesService) Validate(ctx context.Context, params *BeneficiaryCreateParams) (json.RawMessage, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := s.client.post(ctx, beneficiariesBasePath+"/validate", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
