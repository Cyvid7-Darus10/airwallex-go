package airwallex

import (
	"context"
	"encoding/json"
)

const simulationBasePath = "/api/v1/simulation"

// SimulationDepositParams are the parameters for
// SimulationService.CreateDeposit.
type SimulationDepositParams struct {
	Params
	Amount          float64 `json:"amount,omitempty"`
	Currency        string  `json:"currency,omitempty"`
	GlobalAccountID string  `json:"global_account_id,omitempty"`
	Reference       string  `json:"reference,omitempty"`
}

// SimulationTransitionParams drive a resource to its next status, e.g.
// NextStatus: "PAID".
type SimulationTransitionParams struct {
	Params
	NextStatus string `json:"next_status,omitempty"`
}

// SimulationService drives demo-environment state transitions. Every
// method only works against the Demo environment.
type SimulationService struct{ client *Client }

// CreateDeposit simulates money arriving in the wallet.
func (s *SimulationService) CreateDeposit(ctx context.Context, params *SimulationDepositParams) (json.RawMessage, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := s.client.post(ctx, simulationBasePath+"/deposit/create", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SettleDeposit moves a simulated deposit to SETTLED.
func (s *SimulationService) SettleDeposit(ctx context.Context, depositID string) (json.RawMessage, error) {
	return s.depositAction(ctx, depositID, "settle")
}

// RejectDeposit moves a simulated deposit to REJECTED.
func (s *SimulationService) RejectDeposit(ctx context.Context, depositID string) (json.RawMessage, error) {
	return s.depositAction(ctx, depositID, "reject")
}

// ReverseDeposit reverses a simulated deposit.
func (s *SimulationService) ReverseDeposit(ctx context.Context, depositID string) (json.RawMessage, error) {
	return s.depositAction(ctx, depositID, "reverse")
}

func (s *SimulationService) depositAction(ctx context.Context, depositID, action string) (json.RawMessage, error) {
	var result json.RawMessage
	path := simulationBasePath + "/deposits/" + pathEscape(depositID) + "/" + action
	if err := s.client.post(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// TransitionTransfer moves a simulated transfer to its next status.
func (s *SimulationService) TransitionTransfer(ctx context.Context, transferID string, params *SimulationTransitionParams) (json.RawMessage, error) {
	return s.transition(ctx, "transfers", transferID, params)
}

// TransitionPayment moves a simulated payment to its next status.
func (s *SimulationService) TransitionPayment(ctx context.Context, paymentID string, params *SimulationTransitionParams) (json.RawMessage, error) {
	return s.transition(ctx, "payments", paymentID, params)
}

func (s *SimulationService) transition(ctx context.Context, kind, id string, params *SimulationTransitionParams) (json.RawMessage, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	path := simulationBasePath + "/" + kind + "/" + pathEscape(id) + "/transition"
	if err := s.client.post(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
