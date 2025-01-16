package constant_product

import (
	"cosmossdk.io/math"
	"fmt"
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

type ConstantProductTrade func(math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, math.LegacyDec, error)

type FlatPrice struct{}

func (fp FlatPrice) Sell(_, _, v, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	feeAmount := v.Mul(fee)
	return v.Sub(feeAmount), feeAmount, nil
}

func (fp FlatPrice) Buy(_, _, v, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	feeAmount := v.Mul(fee)
	return v.Add(feeAmount), feeAmount, nil
}

func ConstantProductTradeSell(poolFrom, poolTo, offer, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	constantProduct := poolFrom.Mul(poolTo)
	amountToReceive := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer))) // C
	feeAmount := amountToReceive.Mul(fee)
	return amountToReceive.Sub(feeAmount), feeAmount, nil
}

func ConstantProductTradeBuy(poolFrom, poolTo, result, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	if result.GTE(poolTo) {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("requsted amount too large")
	}

	if fee.GTE(math.LegacyOneDec()) {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("fee factor must smaller than 1")
	}

	constantProduct := poolFrom.Mul(poolTo)
	amountToGiveNet := constantProduct.Quo(poolTo.Sub(result)).Sub(poolFrom) // C
	amountToGiveGross := amountToGiveNet.Quo(math.LegacyOneDec().Sub(fee))   // C
	feeAmount := amountToGiveGross.Sub(amountToGiveNet)
	return amountToGiveGross, feeAmount, nil
}

type CalculateMaximumAmount func(math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, error)

func CalculateMaximumGiving(poolFrom, poolTo, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	return poolTo.Mul(maxPrice).Sub(poolFrom), nil
}

func CalculateMaximumReceiving(poolFrom, poolTo, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	if !maxPrice.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("maxPrice must be positive")
	}

	return poolTo.Sub(poolFrom.Quo(maxPrice)), nil // C
}
