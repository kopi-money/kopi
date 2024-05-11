package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/pkg/errors"
)

var (
	KeyCreationFee = []byte("CreationFee")
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	CreationFee = math.NewInt(100_000_000)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		CreationFee: CreationFee,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCreationFee, &p.CreationFee, validateBiggerZero),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateBiggerZero(p.CreationFee); err != nil {
		return errors.Wrap(err, "invalid creation fee")
	}

	return nil
}

func validateBiggerZero(d any) error {
	v, ok := d.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return errors.New("value is nil")
	}

	if v.LT(math.ZeroInt()) {
		return errors.New("share must not be smaller than 0")
	}

	return nil
}
