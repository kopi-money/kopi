package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	BurnThreshold = math.LegacyNewDecWithPrec(9999, 4) // 0.9999
	MintThreshold = math.LegacyOneDec()
	StakingShare  = math.LegacyNewDecWithPrec(1, 1) // 0.1
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		BurnThreshold: BurnThreshold,
		MintThreshold: MintThreshold,
		StakingShare:  StakingShare,
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateGreaterZero(p.MintThreshold); err != nil {
		return fmt.Errorf("invalid mint threshold: %w", err)
	}

	if err := validateGreaterZero(p.BurnThreshold); err != nil {
		return fmt.Errorf("invalid burn threshold: %w", err)
	}

	if err := validateZeroOne(p.StakingShare); err != nil {
		return fmt.Errorf("invalid staking share: %w", err)
	}

	return nil
}

func validateZeroOne(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("share must not be larger than 1")
	}

	if v.IsNegative() {
		return fmt.Errorf("share must not be smaller than 0")
	}

	return nil
}

func validateGreaterZero(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("threshold must not be smaller than 0")
	}

	return nil
}
