package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GetPoolRatio returns the ratio in the form of "One factory denom unit represents x kcoin denom units"
func (lp LiquidityPool) GetPoolRatio() (math.LegacyDec, error) {
	if !lp.FactoryDenomAmount.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("factory denom amount is not positive")
	}

	return lp.KCoinAmount.ToLegacyDec().Quo(lp.FactoryDenomAmount.ToLegacyDec()), nil // C
}

func (lp LiquidityPool) ConvertToKCoin(factoryAmount math.Int) (math.LegacyDec, error) {
	if factoryAmount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	ratio, err := lp.GetPoolRatio()
	if err != nil {
		return math.LegacyDec{}, err
	}

	return ratio.Mul(factoryAmount.ToLegacyDec()), nil
}
