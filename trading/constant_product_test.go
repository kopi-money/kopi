package trading_test

import (
	"github.com/kopi-money/kopi/trading"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestConstantProduct1(t *testing.T) {
	poolSize1 := math.LegacyNewDec(1_000_000)
	poolSize2 := math.LegacyNewDec(1_000_000)
	amount := math.LegacyNewDec(100_000)

	amountToReceive, _, _ := trading.ConstantProductTradeSell(poolSize1, poolSize2, amount, math.LegacyZeroDec())
	require.Equal(t, int64(90909), amountToReceive.TruncateInt().Int64())

	amountToGive, _, err := trading.ConstantProductTradeBuy(poolSize1, poolSize2, amount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(111_111), amountToGive.TruncateInt().Int64())
}

func TestConstantProduct2(t *testing.T) {
	poolSize := math.LegacyNewDec(1_000_000)

	// single trade
	amountGiven1 := math.LegacyNewDec(100_000)
	amountReceived1, _, _ := trading.ConstantProductTradeSell(poolSize, poolSize, amountGiven1, math.LegacyZeroDec())

	// two trades
	amountGiven2 := math.LegacyNewDec(50_000)
	amountReceived2_1, _, _ := trading.ConstantProductTradeSell(poolSize, poolSize, amountGiven2, math.LegacyZeroDec())
	amountReceived2_2, _, _ := trading.ConstantProductTradeSell(poolSize.Add(amountGiven2), poolSize.Sub(amountReceived2_1), amountGiven2, math.LegacyZeroDec())

	require.Equal(t, amountReceived1, amountReceived2_1.Add(amountReceived2_2))
}

func TestConstantProduct3(t *testing.T) {
	poolSize1 := math.LegacyNewDec(1_000_000)
	poolSize2 := math.LegacyNewDec(100_000)
	amountGiven := math.LegacyNewDec(100_000)
	fee := math.LegacyNewDecWithPrec(99, 2)

	// single trade
	amountGivenNet1 := amountGiven.Mul(fee)

	tmpGross, _, _ := trading.ConstantProductTradeSell(poolSize1, poolSize2, amountGiven, math.LegacyZeroDec())
	tmpNet := tmpGross.Mul(fee)
	amountGivenNet2, _, _ := trading.ConstantProductTradeSell(poolSize2.Sub(tmpNet), poolSize1.Add(amountGiven), tmpNet, math.LegacyZeroDec())

	require.Equal(t, amountGivenNet1, amountGivenNet2)
}

func TestCalculateMaximumReceiving1(t *testing.T) {
	poolFrom := math.LegacyNewDec(100)
	poolTo := math.LegacyNewDec(100)
	maxPrice := math.LegacyNewDecWithPrec(11, 1)

	maxAmount, _ := trading.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)

	amountToGive, _, err := trading.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(10), amountToGive.RoundInt64())
}

func TestCalculateMaximumReceiving2(t *testing.T) {
	poolFrom := math.LegacyNewDec(1_000_000_000)
	poolTo := math.LegacyNewDec(1_000_000_000)
	maxPrice := math.LegacyNewDecWithPrec(101, 2)

	maxAmount, _ := trading.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)

	amountToGive, _, err := trading.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000), amountToGive.RoundInt64())

	require.True(t, maxPrice.GTE(amountToGive.Quo(maxAmount))) // C
}

func TestCalculateMaximumGiving1(t *testing.T) {
	X := math.LegacyNewDec(100)
	T1 := math.LegacyNewDec(100)
	T2 := math.LegacyNewDec(100)
	Y := math.LegacyNewDec(100)

	maxPrice := math.LegacyNewDecWithPrec(11, 1)

	maxAmount, _ := trading.CalculateMaximumGivingTwoStep(X, T1, T2, Y, maxPrice)

	out, _, _ := trading.ConstantProductTradeSell(X, T1, maxAmount, math.LegacyZeroDec())
	out, _, _ = trading.ConstantProductTradeSell(T2, Y, out, math.LegacyZeroDec())

	paidPrice := maxAmount.Quo(out)
	require.Equal(t, maxPrice, paidPrice)
}

func TestCalculateMaximumGiving2(t *testing.T) {
	X := math.LegacyNewDec(100)
	T1 := math.LegacyNewDec(100)
	T2 := math.LegacyNewDec(1000)
	Y := math.LegacyNewDec(1000)

	maxPrice := math.LegacyNewDecWithPrec(11, 1)

	maxAmount, _ := trading.CalculateMaximumGivingTwoStep(X, T1, T2, Y, maxPrice)

	out, _, _ := trading.ConstantProductTradeSell(X, T1, maxAmount, math.LegacyZeroDec())
	out, _, _ = trading.ConstantProductTradeSell(T2, Y, out, math.LegacyZeroDec())

	paidPrice := maxAmount.Quo(out)
	require.Equal(t, maxPrice, paidPrice)
}

func TestCalculateMaximumGiving3(t *testing.T) {
	X := math.LegacyNewDec(5000)
	T1 := math.LegacyNewDec(20000)
	T2 := math.LegacyNewDec(400)
	Y := math.LegacyNewDec(100)

	maxPrice := math.LegacyNewDecWithPrec(11, 1)

	maxAmount, _ := trading.CalculateMaximumGivingTwoStep(X, T1, T2, Y, maxPrice)

	out, _, _ := trading.ConstantProductTradeSell(X, T1, maxAmount, math.LegacyZeroDec())
	out, _, _ = trading.ConstantProductTradeSell(T2, Y, out, math.LegacyZeroDec())

	paidPrice := maxAmount.Quo(out)
	require.Equal(t, maxPrice, paidPrice)
}

func TestCalculateMaximumGiving4(t *testing.T) {
	X := math.LegacyNewDec(5000_000000)
	T1 := math.LegacyNewDec(20000_000000)
	T2 := math.LegacyNewDec(280_000000)
	Y := math.LegacyNewDec(70_000000)

	maxPrice, _ := math.LegacyNewDecFromStr("0.504990023568591386")
	maxAmount, _ := trading.CalculateMaximumGivingTwoStep(X, T1, T2, Y, maxPrice)

	out, _, _ := trading.ConstantProductTradeSell(X, T1, maxAmount, math.LegacyZeroDec())
	out, _, _ = trading.ConstantProductTradeSell(T2, Y, out, math.LegacyZeroDec())

	paidPrice := maxAmount.Quo(out)
	require.Equal(t, maxPrice, paidPrice)
}

func TestCalculateSingleMaximumSellableAmount1(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)
	actualTo := math.LegacyNewDec(500)
	virtualTo := math.LegacyNewDec(500)

	maximum := trading.CalculateSingleMaximumSellableAmount(actualFrom, virtualFrom, actualTo, virtualTo)

	fullFrom := actualFrom.Add(virtualFrom)
	fullTo := actualTo.Add(virtualTo)
	amount, _, _ := trading.ConstantProductTradeSell(fullFrom, fullTo, maximum.ToLegacyDec(), math.LegacyZeroDec())

	require.Equal(t, actualTo.TruncateInt64(), amount.TruncateInt64())
}
