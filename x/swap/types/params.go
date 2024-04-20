package types

import (
	"cosmossdk.io/math"
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/pkg/errors"
)

var (
	KeyStakingShare = []byte("StakingShare")
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	StakingShare = math.LegacyNewDecWithPrec(1, 1) // 0.1
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		StakingShare: StakingShare,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyStakingShare, &p.StakingShare, validateZeroOne),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateZeroOne(p.StakingShare); err != nil {
		return errors.Wrap(err, "invalid staking share")
	}

	return nil
}

func validateZeroOne(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return errors.New("value is nil")
	}

	if v.GT(math.LegacyOneDec()) {
		return errors.New("share must not be larger than 1")
	}

	if v.LT(math.LegacyZeroDec()) {
		return errors.New("share must be smaller than 0")
	}

	return nil
}
