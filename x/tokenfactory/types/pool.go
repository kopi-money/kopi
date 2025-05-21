package types

import "cosmossdk.io/math"

func (lp LiquidityPool) GetAmounts(share math.LegacyDec) (math.Int, math.Int) {
	amountFactory := lp.FactoryDenomAmount.ToLegacyDec().Mul(share).TruncateInt()
	amountToken := lp.KCoinAmount.ToLegacyDec().Mul(share).TruncateInt()
	return amountFactory, amountToken
}

func (lp LiquidityPool) Price() (math.LegacyDec, error) {
	if !lp.KCoinAmount.IsPositive() {
		return math.LegacyDec{}, ErrNegativeLiquidity
	}

	return lp.FactoryDenomAmount.ToLegacyDec().Quo(lp.KCoinAmount.ToLegacyDec()), nil
}
