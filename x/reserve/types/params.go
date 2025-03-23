package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	KCoinBurnShare                    = math.LegacyOneDec()
	SellThreshold                     = math.LegacyOneDec()
	BuyThreshold                      = math.LegacyNewDecWithPrec(9999, 4)
	TradeFeeBaseIncomeShareToStakers  = math.LegacyNewDecWithPrec(1, 1)
	TradeFeeOtherIncomeShareToStakers = math.LegacyNewDecWithPrec(1, 1)
)

func DefaultParams() Params {
	return Params{
		KcoinBurnShare:                    KCoinBurnShare,
		SellThreshold:                     SellThreshold,
		BuyThreshold:                      BuyThreshold,
		TradeFeeBaseIncomeShareToStakers:  TradeFeeBaseIncomeShareToStakers,
		TradeFeeOtherIncomeShareToStakers: TradeFeeOtherIncomeShareToStakers,
	}
}

func NewParams() Params {
	return DefaultParams()
}

func (p Params) Validate() error {
	if p.KcoinBurnShare.IsNil() {
		return fmt.Errorf("kcoinburn share must not be nil")
	}

	if p.KcoinBurnShare.IsNegative() {
		return fmt.Errorf("kcoin burn share must not be below 0")
	}

	if p.KcoinBurnShare.GT(math.LegacyOneDec()) {
		return fmt.Errorf("kcoin burn share must not be larger than 1")
	}

	if p.SellThreshold.IsNil() {
		return fmt.Errorf("sell threshold must not be nil")
	}

	if p.SellThreshold.LT(math.LegacyOneDec()) {
		return fmt.Errorf("sell threshold must not be less than 1")
	}

	if p.BuyThreshold.IsNil() {
		return fmt.Errorf("buy threshold must not be nil")
	}

	if p.BuyThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("buy threshold must not be larger than 1")
	}

	if p.TradeFeeBaseIncomeShareToStakers.IsNil() {
		p.TradeFeeBaseIncomeShareToStakers = TradeFeeBaseIncomeShareToStakers
	}

	if err := validateBetweenZeroAndOne(p.TradeFeeBaseIncomeShareToStakers); err != nil {
		return fmt.Errorf("invalid trade fee base income share to stakers: %w", err)
	}

	if p.TradeFeeOtherIncomeShareToStakers.IsNil() {
		p.TradeFeeOtherIncomeShareToStakers = TradeFeeOtherIncomeShareToStakers
	}

	if err := validateBetweenZeroAndOne(p.TradeFeeOtherIncomeShareToStakers); err != nil {
		return fmt.Errorf("invalid trade fee other income share to stakers: %w", err)
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
