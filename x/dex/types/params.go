package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	TradeFee              = math.LegacyNewDecWithPrec(1, 3)      // 0.001 -> 0.1%
	OrderFee              = math.LegacyNewDecWithPrec(5, 3)      // 0.005 -> 0.5%
	ReserveShare          = math.LegacyNewDecWithPrec(5, 1)      // 0.5 -> 50%
	VirtualLiquidityDecay = math.LegacyNewDecWithPrec(999997, 6) // 0.999997
	TradeAmountDecay      = math.LegacyNewDecWithPrec(95, 2)     // 0.95
	MaxOrderLife          = 60 * 60 * 24 * 7
	DiscountLevels        = []DiscountLevel{
		{
			TradeAmount: math.LegacyNewDec(1_000_000),
			Discount:    math.LegacyNewDecWithPrec(1, 2),
		},
		{
			TradeAmount: math.LegacyNewDec(10_000_000),
			Discount:    math.LegacyNewDecWithPrec(1, 1),
		},
	}
	TradeBaseValue = math.LegacyNewDec(1_000000_000000) // 1mio
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		TradeFee:              TradeFee,
		OrderFee:              OrderFee,
		VirtualLiquidityDecay: VirtualLiquidityDecay,
		ReserveShare:          ReserveShare,
		MaxOrderLife:          uint64(MaxOrderLife),
		TradeAmountDecay:      TradeAmountDecay,
		DiscountLevels:        DiscountLevels,
		TradeBaseValue:        TradeBaseValue,
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := p.validateDiscountLevels(); err != nil {
		return fmt.Errorf("invalid discount level: %w", err)
	}

	if err := validateZeroOne(p.TradeFee); err != nil {
		return fmt.Errorf("invalid trade fee: %w", err)
	}

	if err := validateZeroOne(p.OrderFee); err != nil {
		return fmt.Errorf("invalid order fee: %w", err)
	}

	if err := validateVirtualLiquidityDecay(p.VirtualLiquidityDecay); err != nil {
		return fmt.Errorf("invalid virtual liquidity decay: %w", err)
	}

	if err := validateZeroOne(p.ReserveShare); err != nil {
		return fmt.Errorf("invalid reserve share: %w", err)
	}

	if err := validateBiggerThanZero(p.MaxOrderLife); err != nil {
		return fmt.Errorf("invalid fee reimbursement: %w", err)
	}

	if err := validateBetweenZeroAndOne(p.TradeAmountDecay); err != nil {
		return fmt.Errorf("invalid trade amount decay: %w", err)
	}

	if p.TradeBaseValue.IsNil() {
		p.TradeBaseValue = TradeBaseValue
	}

	if err := validateDexBiggerThanZero(p.TradeBaseValue); err != nil {
		return fmt.Errorf("invalid trade base value: %w", err)
	}

	return nil
}

func (p Params) validateDiscountLevels() error {
	for index, discountLevel := range p.DiscountLevels {
		if err := validateBetweenZeroAndOne(discountLevel.Discount); err != nil {
			return fmt.Errorf("invalid discount for entry with index %v: %w", index, err)
		}

		if discountLevel.TradeAmount.IsZero() {
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
		return fmt.Errorf("value is nil")
	}

	if v.GTE(math.LegacyOneDec()) {
		return fmt.Errorf("fee must not be larger than 1")
	}

	if v.IsNegative() {
		return fmt.Errorf("fee must be bigger than 0")
	}

	return nil
}

func validateVirtualLiquidityDecay(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("must not be larger than 1")
	}

	if !v.IsPositive() {
		return fmt.Errorf("must be bigger than 0")
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
		return fmt.Errorf("fee must not be larger than 1")
	}

	if v.LTE(math.LegacyZeroDec()) {
		return fmt.Errorf("fee must be bigger than 0")
	}

	return nil
}

func validateBiggerThanZero(d any) error {
	v, ok := d.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v < 1 {
		return fmt.Errorf("value is smaller than 1")
	}

	return nil
}

func validateDexBiggerThanZero(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if !v.IsPositive() {
		return fmt.Errorf("value is smaller than 1")
	}

	return nil
}

func validateBetweenZeroAndOne(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if !v.IsPositive() {
		return fmt.Errorf("value has to be bigger than 0")
	}

	if !v.LT(math.LegacyOneDec()) {
		return fmt.Errorf("value has to be less than 1")
	}

	return nil
}
