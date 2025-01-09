package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	CreationFee     = math.NewInt(1_000_000)
	ReserveFee      = math.LegacyNewDecWithPrec(1, 3)
	PoolFee         = math.LegacyNewDecWithPrec(1, 3)
	MinimumPoolSize = math.NewInt(1_000_000_000)
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		CreationFee:     CreationFee,
		ReserveFee:      ReserveFee,
		MinimumPoolSize: MinimumPoolSize,
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.CreationFee.IsNil() {
		p.CreationFee = CreationFee
	}

	if p.ReserveFee.IsNil() {
		p.ReserveFee = ReserveFee
	}

	if p.MinimumPoolSize.IsNil() {
		p.MinimumPoolSize = MinimumPoolSize
	}

	if err := validateBiggerZero(p.CreationFee); err != nil {
		return fmt.Errorf("invalid creation fee: %w", err)
	}

	if err := validateTradeFee(p.ReserveFee); err != nil {
		return fmt.Errorf("invalid reserve fee: %w", err)
	}

	if err := validateBiggerZero(p.MinimumPoolSize); err != nil {
		return fmt.Errorf("invalid minimum pool size: %w", err)
	}

	return nil
}

func validateTradeFee(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("fee must not be smaller than 0")
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("fee must not be greater than 1")
	}

	return nil
}

func validateBiggerZero(d any) error {
	v, ok := d.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("share must not be smaller than 0")
	}

	return nil
}
