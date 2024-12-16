package types

import (
	"fmt"
	
	"cosmossdk.io/math"
)

var (
	kCoinBurnShare = math.LegacyOneDec()
)

func DefaultParams() Params {
	return Params{
		KcoinBurnShare: kCoinBurnShare,
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

	return nil
}
