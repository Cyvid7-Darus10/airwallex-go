package airwallex

import (
	"context"
	"iter"
)

const walletTransfersBasePath = "/api/v1/wallet_transfers"

// WalletTransferBeneficiary identifies the receiving wallet.
type WalletTransferBeneficiary struct {
	AccountName   string `json:"account_name,omitempty"`
	AccountNumber string `json:"account_number,omitempty"`
}

// WalletTransfer is a transfer between Airwallex wallets
// (/api/v1/wallet_transfers).
type WalletTransfer struct {
	APIResource
	WalletTransferID string `json:"wallet_transfer_id"`
	RequestID        string `json:"request_id"`
	ShortReferenceID string `json:"short_reference_id"`
	Status           string `json:"status"`

	TransferAmount   float64                    `json:"transfer_amount"`
	TransferCurrency string                     `json:"transfer_currency"`
	Beneficiary      *WalletTransferBeneficiary `json:"beneficiary"`

	Reason    string `json:"reason"`
	Reference string `json:"reference"`

	CreatedAt string `json:"created_at"`
	SettledAt string `json:"settled_at"`
}

// WalletTransferCreateParams are the parameters for
// WalletTransfersService.Create.
type WalletTransferCreateParams struct {
	Params
	// RequestID makes the create idempotent; auto-generated when empty.
	RequestID        string                     `json:"request_id,omitempty"`
	TransferAmount   float64                    `json:"transfer_amount,omitempty"`
	TransferCurrency string                     `json:"transfer_currency,omitempty"`
	Beneficiary      *WalletTransferBeneficiary `json:"beneficiary,omitempty"`
	Reason           string                     `json:"reason,omitempty"`
	Reference        string                     `json:"reference,omitempty"`
}

// WalletTransferListParams filter WalletTransfersService.List.
type WalletTransferListParams struct {
	ListParams
	Status           string `json:"status,omitempty"`
	TransferCurrency string `json:"transfer_currency,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	ShortReferenceID string `json:"short_reference_id,omitempty"`
	FromCreatedAt    string `json:"from_created_at,omitempty"`
	ToCreatedAt      string `json:"to_created_at,omitempty"`
}

// WalletTransfersService moves money between Airwallex wallets.
type WalletTransfersService struct{ client *Client }

// Create creates a wallet transfer. A request_id is generated automatically
// when params.RequestID is empty, making the call idempotent.
func (s *WalletTransfersService) Create(ctx context.Context, params *WalletTransferCreateParams) (*WalletTransfer, error) {
	body, err := idempotentBody(params)
	if err != nil {
		return nil, err
	}
	transfer := &WalletTransfer{}
	if err := s.client.post(ctx, walletTransfersBasePath+"/create", body, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// Retrieve fetches a single wallet transfer by id.
func (s *WalletTransfersService) Retrieve(ctx context.Context, walletTransferID string) (*WalletTransfer, error) {
	transfer := &WalletTransfer{}
	if err := s.client.get(ctx, walletTransfersBasePath+"/"+pathEscape(walletTransferID), nil, transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// List returns one page of wallet transfers, filtered by params (may be nil).
func (s *WalletTransfersService) List(ctx context.Context, params *WalletTransferListParams) (*Page[WalletTransfer], error) {
	return listPage[WalletTransfer](ctx, s.client, walletTransfersBasePath, params)
}

// All iterates every wallet transfer across every page, fetching lazily.
func (s *WalletTransfersService) All(ctx context.Context, params *WalletTransferListParams) iter.Seq2[WalletTransfer, error] {
	page, err := s.List(ctx, params)
	return iterPages(ctx, page, err)
}
