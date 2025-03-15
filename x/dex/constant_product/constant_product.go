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

type CalculateMaximumAmountOneStep func(math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, error)

func CalculateMaximumGivingOneStep(poolFrom, poolTo, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	return poolTo.Mul(maxPrice).Sub(poolFrom), nil
}

func CalculateMaximumReceiving(poolFrom, poolTo, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	if !maxPrice.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("maxPrice must be positive")
	}

	return poolTo.Sub(poolFrom.Quo(maxPrice)), nil // C
}

type CalculateMaximumAmountTwoStep func(math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec) (math.LegacyDec, error)

func CalculateMaximumGivingTwoStep(X, T1, T2, Y, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	//if T1.Equal(T2) {
	//	return CalculateMaximumGivingOneStep(X, Y, maxPrice)
	//}

	p1 := maxPrice.Mul(Y).Mul(T1)
	p2 := X.Mul(T2)
	nominator := p1.Sub(p2)
	denominator := T1.Add(T2)
	return nominator.Quo(denominator), nil
}

func CalculateMaximumReceivingTwoStep(X, T1, T2, Y, maxPrice math.LegacyDec) (math.LegacyDec, error) {
	if T1.Equal(T2) {
		return CalculateMaximumReceiving(X, Y, maxPrice)
	}

	A := maxPrice.Mul(X.Add(T2))
	B := X.Mul(X.Add(T2)).Sub(X.Mul(T1)).Sub(maxPrice.Mul(T2).Mul(T2.Add(X).Sub(Y)))
	C := X.Mul(T1).Mul(T2).Sub(X.Mul(T2).Mul(T2.Add(X).Sub(Y)))

	tmp := B.Mul(B).Sub(math.LegacyNewDec(4).Mul(A).Mul(C))
	if tmp.IsNegative() {
		return math.LegacyDec{}, fmt.Errorf("no solution")
	}

	if tmp.IsZero() {
		return B.Quo(math.LegacyNewDec(2).Mul(A)).Neg(), nil
	}

	root, err := tmp.ApproxSqrt()
	if err != nil {
		return math.LegacyDec{}, err
	}

	nominator := root.Sub(B)
	denominator := math.LegacyNewDec(2).Mul(A)
	return nominator.Quo(denominator), nil
}

func CalculateSingleMaximumSellableAmount(actualFrom, virtualFrom, actualTo, virtualTo math.LegacyDec) *math.Int {
	if virtualTo.IsNil() || virtualTo.IsZero() {
		return nil
	}

	X := actualFrom.Add(virtualFrom)
	maximum := X.Mul(actualTo.Quo(virtualTo)) // C
	maximumInt := maximum.TruncateInt()
	return &maximumInt
}
