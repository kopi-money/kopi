package utils

import (
	"math"

	sdkmath "cosmossdk.io/math"
)

func MovingAverage(amountOld, amountNew sdkmath.LegacyDec) sdkmath.LegacyDec {
	facNew := sdkmath.LegacyOneDec().Quo(sdkmath.LegacyNewDec(int64(BlocksPerDay)))
	facOld := sdkmath.LegacyOneDec().Sub(facNew)

	sum := sdkmath.LegacyZeroDec()
	sum = sum.Add(amountOld.Mul(facOld))
	sum = sum.Add(amountNew.Mul(facNew))
	return sum
}

func MinMax(amount, min, max sdkmath.LegacyDec) sdkmath.LegacyDec {
	amount = sdkmath.LegacyMaxDec(amount, min)
	amount = sdkmath.LegacyMinDec(amount, max)
	return amount
}

func pow(amount int64) int64 {
	fac := int64(math.Pow(10, float64(DecimalPlaces)))
	return amount * fac
}

func PowInt(amount int64) sdkmath.Int {
	return sdkmath.NewInt(pow(amount))
}

func PowDec(amount int64) sdkmath.LegacyDec {
	return sdkmath.LegacyNewDec(pow(amount))
}
