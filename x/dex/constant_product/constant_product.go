package constant_product

import (
	"cosmossdk.io/math"
	"fmt"
)

type ConstantProduct struct{}

func (cp ConstantProduct) Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int {
	constantProduct := poolFrom.Mul(poolTo)
	amount := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer)))
	return amount.TruncateInt()
}

func (cp ConstantProduct) Backward(poolFrom, poolTo, result math.LegacyDec) math.Int {
	constantProduct := poolFrom.Mul(poolTo)
	return constantProduct.Quo(poolTo.Sub(result)).Sub(poolFrom).TruncateInt()
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
	amountToReceive := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer)))
	feeAmount := amountToReceive.Mul(fee)
	return amountToReceive.Sub(feeAmount), feeAmount, nil
}

func ConstantProductTradeBuy(poolFrom, poolTo, result, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	if result.GTE(poolTo) {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("requsted amount too large")
	}

	constantProduct := poolFrom.Mul(poolTo)
	//amountToGive := poolFrom.Sub(constantProduct.Quo(poolTo.Add(result)))
	amountToGive := constantProduct.Quo(poolTo.Sub(result)).Sub(poolFrom)
	feeAmount := amountToGive.Mul(fee)
	return amountToGive.Add(feeAmount), feeAmount, nil
}

type CalculateMaximumAmount func(math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec) math.LegacyDec

func CalculateMaximumGiving(poolFrom, poolTo, maxPrice, fee math.LegacyDec) math.LegacyDec {
	maxPrice = maxPrice.Mul(math.LegacyOneDec().Sub(fee))
	return poolTo.Mul(maxPrice).Sub(poolFrom)
}

func CalculateMaximumReceiving(poolFrom, poolTo, maxPrice, fee math.LegacyDec) math.LegacyDec {
	maxPrice = maxPrice.Mul(math.LegacyOneDec().Sub(fee))
	return maxPrice.Mul(poolTo).Sub(poolFrom).Quo(maxPrice)
}
