package types

import (
	"cosmossdk.io/math"
	"errors"
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/kopi-money/kopi/utils"
)

var (
	KeyTradeFee              = []byte("TradeFee")
	KeyVirtualLiquidityDecay = []byte("VirtualLiquidityDecay")
	KeyReserveShare          = []byte("ReserveShare")
	KeyFeeReimbursement      = []byte("FeeReimbursement")
	KeyMaxOrderLife          = []byte("MaxOrderLife")
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	FeeReimbursement      = math.LegacyNewDecWithPrec(5, 1)      // 0.5
	TradeFee              = math.LegacyNewDecWithPrec(1, 3)      // 0.001 -> 0.1%
	ReserveShare          = math.LegacyNewDecWithPrec(5, 1)      // 0.5 -> 50%
	VirtualLiquidityDecay = math.LegacyNewDecWithPrec(999997, 6) // 0.999997
	MaxOrderLife          = utils.BlocksPerDay * 7
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		TradeFee:              TradeFee,
		VirtualLiquidityDecay: VirtualLiquidityDecay,
		ReserveShare:          ReserveShare,
		FeeReimbursement:      FeeReimbursement,
		MaxOrderLife:          MaxOrderLife,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyTradeFee, &p.TradeFee, validateZeroOne),
		paramtypes.NewParamSetPair(KeyVirtualLiquidityDecay, &p.VirtualLiquidityDecay, validateZeroOne),
		paramtypes.NewParamSetPair(KeyReserveShare, &p.ReserveShare, validateZeroOne),
		paramtypes.NewParamSetPair(KeyFeeReimbursement, &p.FeeReimbursement, validateLessThanOne),
		paramtypes.NewParamSetPair(KeyMaxOrderLife, &p.MaxOrderLife, validateBiggerThanZero),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	return nil
}

func validateLessThanOne(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return errors.New("value is nil")
	}

	if v.GTE(math.LegacyOneDec()) {
		return errors.New("fee must not be larger than 1")
	}

	if v.LT(math.LegacyZeroDec()) {
		return errors.New("fee must be bigger than 0")
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

	if v.LTE(math.LegacyZeroDec()) {
		return errors.New("fee must be bigger than 0")
	}

	return nil
}

func validateBiggerThanZero(d any) error {
	v, ok := d.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v < 1 {
		return errors.New("value is smaller than 1")
	}

	return nil
}
