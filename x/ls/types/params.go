package types

import (
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var topNValidators int64 = 20

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		TopNValidators: topNValidators,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.TopNValidators < 1 {
		return fmt.Errorf("TopNValidators must be at least 1, got %d", p.TopNValidators)
	}

	return nil
}
