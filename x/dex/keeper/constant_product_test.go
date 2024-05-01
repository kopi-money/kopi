package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
)

func TestConstantProduct1(t *testing.T) {
	poolSize := math.LegacyNewDec(1_000_000)

	// single trade
	amountGiven1 := math.LegacyNewDec(100_000)
	amountReceived1 := dexkeeper.ConstantProductTrade(poolSize, poolSize, amountGiven1)

	// two trades
	amountGiven2 := math.LegacyNewDec(50_000)
	amountReceived2_1 := dexkeeper.ConstantProductTrade(poolSize, poolSize, amountGiven2)
	amountReceived2_2 := dexkeeper.ConstantProductTrade(poolSize.Add(amountGiven2), poolSize.Sub(amountReceived2_1), amountGiven2)

	require.Equal(t, amountReceived1, amountReceived2_1.Add(amountReceived2_2))
}

func TestConstantProduct2(t *testing.T) {
	poolSize1 := math.LegacyNewDec(1_000_000)
	poolSize2 := math.LegacyNewDec(100_000)
	amountGiven := math.LegacyNewDec(100_000)
	fee := math.LegacyNewDecWithPrec(99, 2)

	// single trade
	amountGivenNet1 := amountGiven.Mul(fee)

	tmpGross := dexkeeper.ConstantProductTrade(poolSize1, poolSize2, amountGiven)
	tmpNet := tmpGross.Mul(fee)
	amountGivenNet2 := dexkeeper.ConstantProductTrade(poolSize2.Sub(tmpNet), poolSize1.Add(amountGiven), tmpNet)

	require.Equal(t, amountGivenNet1, amountGivenNet2)
}
