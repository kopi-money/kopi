package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/pkg/errors"
)

var (
	KeyCollateralDiscount  = []byte("CollateralDiscount")
	KeyMinRedemptionFee    = []byte("MinRedemptionFee")
	KeyMinimumInterestRate = []byte("MinimumInterestRate")
	KeyA                   = []byte("A")
	KeyB                   = []byte("B")
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	CollateralDiscount  = math.LegacyNewDecWithPrec(95, 2) // 0.95
	ProtocolShare       = math.LegacyNewDecWithPrec(5, 1)  // 0.5
	MinRedemptionFee    = math.LegacyNewDecWithPrec(1, 2)  // 0.01
	MinimumInterestRate = math.LegacyNewDecWithPrec(5, 2)  // 0.05
	A                   = math.LegacyNewDec(12)
	B                   = math.LegacyNewDec(131072)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		CollateralDiscount: CollateralDiscount,
		ProtocolShare:      ProtocolShare,
		MinRedemptionFee:   MinRedemptionFee,
		MinInterestRate:    MinimumInterestRate,
		A:                  A,
		B:                  B,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCollateralDiscount, &p.CollateralDiscount, validateZeroOne),
		paramtypes.NewParamSetPair(KeyMinRedemptionFee, &p.MinRedemptionFee, validateZeroOne),
		paramtypes.NewParamSetPair(KeyMinimumInterestRate, &p.MinInterestRate, validateZeroOne),
		paramtypes.NewParamSetPair(KeyA, &p.A, validateNumber),
		paramtypes.NewParamSetPair(KeyB, &p.B, validateNumber),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateZeroOne(p.CollateralDiscount); err != nil {
		return errors.Wrap(err, "invalid collateral discount")
	}

	if err := validateZeroOne(p.MinRedemptionFee); err != nil {
		return errors.Wrap(err, "invalid minimum redemption fee")
	}

	if err := validateZeroOne(p.MinInterestRate); err != nil {
		return errors.Wrap(err, "invalid minimum interest rate")
	}

	if err := validateNumber(p.A); err != nil {
		return errors.Wrap(err, "invalid A")
	}

	if err := validateNumber(p.B); err != nil {
		return errors.Wrap(err, "invalid B")
	}

	return nil
}

func validateNumber(d any) error {
	_, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
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
		return errors.New("fee must not be larger than 1")
	}

	if v.LT(math.LegacyZeroDec()) {
		return errors.New("fee must be smaller than 0")
	}

	return nil
}
