package types

import (
	"cosmossdk.io/math"
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/pkg/errors"
)

var (
	KeyTradeFee              = []byte("TradeFee")
	KeyVirtualLiquidityDecay = []byte("VirtualLiquidityDecay")
	KeyReserveShare          = []byte("ReserveShare")
	KeyFeeReimbursement      = []byte("FeeReimbursement")
	KeyMaxOrderLife          = []byte("MaxOrderLife")
	KeyTradeAmountDecay      = []byte("KeyTradeAmountDecay")
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	FeeReimbursement      = math.LegacyNewDecWithPrec(5, 1)      // 0.5
	TradeFee              = math.LegacyNewDecWithPrec(1, 3)      // 0.001 -> 0.1%
	ReserveShare          = math.LegacyNewDecWithPrec(5, 1)      // 0.5 -> 50%
	VirtualLiquidityDecay = math.LegacyNewDecWithPrec(999997, 6) // 0.999997
	TradeAmountDecay      = math.LegacyNewDecWithPrec(95, 2)     // 0.95
	MaxOrderLife          = utils.BlocksPerDay * 7
	DiscountLevels        = []*DiscountLevel{
		{
			TradeAmount: math.LegacyNewDec(1_000_000),
			Discount:    math.LegacyNewDecWithPrec(1, 2),
		},
		{
			TradeAmount: math.LegacyNewDec(10_000_000),
			Discount:    math.LegacyNewDecWithPrec(1, 1),
		},
	}
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
		TradeAmountDecay:      TradeAmountDecay,
		DiscountLevels:        DiscountLevels,
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
		paramtypes.NewParamSetPair(KeyTradeAmountDecay, &p.TradeAmountDecay, validateBetweenZeroAndOne),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := p.validateDiscountLevels(); err != nil {
		return errors.Wrap(err, "invalid discount level")
	}

	if err := validateZeroOne(p.TradeFee); err != nil {
		return errors.Wrap(err, "invalid trade fee")
	}

	if err := validateZeroOne(p.VirtualLiquidityDecay); err != nil {
		return errors.Wrap(err, "invalid virtual liquidity decay")
	}

	if err := validateZeroOne(p.ReserveShare); err != nil {
		return errors.Wrap(err, "invalid reserve share")
	}

	if err := validateLessThanOne(p.FeeReimbursement); err != nil {
		return errors.Wrap(err, "invalid fee reimbursement")
	}

	if err := validateBiggerThanZero(p.MaxOrderLife); err != nil {
		return errors.Wrap(err, "invalid fee reimbursement")
	}

	if err := validateBetweenZeroAndOne(p.TradeAmountDecay); err != nil {
		return errors.Wrap(err, "invalid trade amount decay")
	}

	return nil
}

func (p Params) validateDiscountLevels() error {
	for index, discountLevel := range p.DiscountLevels {
		if err := validateBetweenZeroAndOne(discountLevel.Discount); err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid discount for entry with index %v", index))
		}

		if discountLevel.TradeAmount.Equal(math.LegacyZeroDec()) {
			return fmt.Errorf("trade amount for entry with index %v must not be zero", index)
		}
	}

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

func validateBetweenZeroAndOne(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return errors.New("value is nil")
	}

	if !v.GT(math.LegacyZeroDec()) {
		return errors.New("value has to be bigger than 0")
	}

	if !v.LT(math.LegacyOneDec()) {
		return errors.New("value has to be less than 1")
	}

	return nil
}
