package constant_product_test

import (
	"testing"

	"github.com/kopi-money/kopi/x/dex/constant_product"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestConstantProduct1(t *testing.T) {
	poolSize1 := math.LegacyNewDec(1_000_000)
	poolSize2 := math.LegacyNewDec(1_000_000)
	amount := math.LegacyNewDec(100_000)

	amountToReceive, _, _ := constant_product.ConstantProductTradeSell(poolSize1, poolSize2, amount, math.LegacyZeroDec())
	require.Equal(t, int64(90909), amountToReceive.TruncateInt().Int64())

	amountToGive, _, err := constant_product.ConstantProductTradeBuy(poolSize1, poolSize2, amount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(111_111), amountToGive.TruncateInt().Int64())
}

func TestConstantProduct2(t *testing.T) {
	poolSize := math.LegacyNewDec(1_000_000)

	// single trade
	amountGiven1 := math.LegacyNewDec(100_000)
	amountReceived1, _, _ := constant_product.ConstantProductTradeSell(poolSize, poolSize, amountGiven1, math.LegacyZeroDec())

	// two trades
	amountGiven2 := math.LegacyNewDec(50_000)
	amountReceived2_1, _, _ := constant_product.ConstantProductTradeSell(poolSize, poolSize, amountGiven2, math.LegacyZeroDec())
	amountReceived2_2, _, _ := constant_product.ConstantProductTradeSell(poolSize.Add(amountGiven2), poolSize.Sub(amountReceived2_1), amountGiven2, math.LegacyZeroDec())

	require.Equal(t, amountReceived1, amountReceived2_1.Add(amountReceived2_2))
}

func TestConstantProduct3(t *testing.T) {
	poolSize1 := math.LegacyNewDec(1_000_000)
	poolSize2 := math.LegacyNewDec(100_000)
	amountGiven := math.LegacyNewDec(100_000)
	fee := math.LegacyNewDecWithPrec(99, 2)

	// single trade
	amountGivenNet1 := amountGiven.Mul(fee)

	tmpGross, _, _ := constant_product.ConstantProductTradeSell(poolSize1, poolSize2, amountGiven, math.LegacyZeroDec())
	tmpNet := tmpGross.Mul(fee)
	amountGivenNet2, _, _ := constant_product.ConstantProductTradeSell(poolSize2.Sub(tmpNet), poolSize1.Add(amountGiven), tmpNet, math.LegacyZeroDec())

	require.Equal(t, amountGivenNet1, amountGivenNet2)
}

func TestCalculateMaximumReceiving1(t *testing.T) {
	poolFrom := math.LegacyNewDec(100)
	poolTo := math.LegacyNewDec(100)
	maxPrice := math.LegacyNewDecWithPrec(11, 1)

	maxAmount := constant_product.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)

	amountToGive, _, err := constant_product.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(10), amountToGive.RoundInt64())
}

func TestCalculateMaximumReceiving2(t *testing.T) {
	poolFrom := math.LegacyNewDec(1_000_000_000)
	poolTo := math.LegacyNewDec(1_000_000_000)
	maxPrice := math.LegacyNewDecWithPrec(101, 2)

	maxAmount := constant_product.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)

	amountToGive, _, err := constant_product.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000), amountToGive.RoundInt64())

	require.True(t, maxPrice.GTE(amountToGive.Quo(maxAmount)))
}
