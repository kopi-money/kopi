package trading

import (
	"fmt"

	"cosmossdk.io/math"
)

type ConstantProduct struct{}

func (cp ConstantProduct) Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int {
	constantProduct := poolFrom.Mul(poolTo)
	amount := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer))) // C
	return amount.TruncateInt()
}

func (cp ConstantProduct) Backward(poolFrom, poolTo, requested math.LegacyDec) (math.Int, error) {
	if requested.GTE(poolTo) {
		return math.ZeroInt(), fmt.Errorf("requested value must not be GTE poolTo")
	}

	constantProduct := poolFrom.Mul(poolTo)
	return constantProduct.Quo(poolTo.Sub(requested)).Sub(poolFrom).TruncateInt(), nil // C
}

type ConstantProductTrade func(math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, error)
type FlatPrice func(math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, error)

func FlatSell(liqFrom, liqTo, v math.LegacyDec) (math.LegacyDec, error) {
	price := liqFrom.Quo(liqTo)
	return v.Quo(price), nil
}

func FlatBuy(liqFrom, liqTo, v math.LegacyDec) (math.LegacyDec, error) {
	price := liqFrom.Quo(liqTo)
	return v.Mul(price), nil
}

func FlatTrade(_, _, v math.LegacyDec) (math.LegacyDec, error) {
	return v, nil
}

func ConstantProductTradeSell(poolFrom, poolTo, offer math.LegacyDec) (math.LegacyDec, error) {
	constantProduct := poolFrom.Mul(poolTo)
	amountToReceive := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer))) // C
	return amountToReceive, nil
}

func ConstantProductTradeBuy(poolFrom, poolTo, requested math.LegacyDec) (math.LegacyDec, error) {
	if requested.GTE(poolTo) {
		fmt.Println(requested)
		fmt.Println(poolTo)

		return math.LegacyDec{}, ErrRequestedAmountTooLarge
	}

	constantProduct := poolFrom.Mul(poolTo)
	amountToGive := constantProduct.Quo(poolTo.Sub(requested)).Sub(poolFrom) // C
	return amountToGive, nil
}

type CalculateMaximumAmountByPrice func(math.LegacyDec, math.LegacyDec, math.LegacyDec) math.LegacyDec

func CalculateMaximumSellableByPrice(poolFrom, poolTo, maxPrice math.LegacyDec) math.LegacyDec {
	return poolTo.Mul(maxPrice).Sub(poolFrom)
}

func CalculateMaximumBuyableByPrice(poolFrom, poolTo, maxPrice math.LegacyDec) math.LegacyDec {
	return poolTo.Sub(poolFrom.Quo(maxPrice)) // C
}

type CalculateMaximumAmountByLiquidity func(Liquidity, Liquidity) *math.Int

func CalculateMaximumSellableByLiquidity(liqFrom, liqTo Liquidity) *math.Int {
	if liqTo.Virtual.IsNil() || liqTo.Virtual.IsZero() {
		return nil
	}

	X := liqFrom.Full()
	maximum := X.Mul(liqTo.Actual.Quo(liqTo.Virtual)) // C
	maximumInt := maximum.TruncateInt()
	return &maximumInt
}

func CalculateMaximumBuyableByLiquidity(_, liqTo Liquidity) *math.Int {
	maximum := liqTo.Actual
	if liqTo.Virtual.IsNil() || liqTo.Actual.IsZero() {
		maximum = maximum.Sub(math.LegacyOneDec())
	}

	maximumInt := maximum.TruncateInt()
	return &maximumInt
}

func CalculateMaximumBuyableByWallet(liqFrom, liqTo Liquidity, a math.Int) math.Int {
	X := liqFrom.Full()
	Y := liqTo.Full()
	k := X.Mul(Y)

	return Y.Sub(k.Quo(X.Add(a.ToLegacyDec()))).TruncateInt()
}
