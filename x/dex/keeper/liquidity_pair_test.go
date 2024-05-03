package keeper_test

import (
	"github.com/kopi-money/kopi/testutil"
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/stretchr/testify/require"
)

func TestLiquidityPairs1(t *testing.T) {
	testutil.RepeatTest(t, func() []byte {
		k, msg, ctx := keepertest.SetupDexMsgServer(t)

		require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))
		liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		require.True(t, found)
		require.Equal(t, math.NewInt(1), liq)

		ratios := k.GetAllRatio(ctx)
		require.True(t, len(ratios) > 0)

		r, found := k.GetRatio(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *r.Ratio)

		pair, found := k.GetLiquidityPair(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.LegacyZeroDec(), pair.VirtualBase)
		require.Equal(t, math.LegacyNewDecWithPrec(1, 1), pair.VirtualOther)

		require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))
		liq, found = k.GetLiquiditySum(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.NewInt(1), liq)

		pair, found = k.GetLiquidityPair(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.LegacyNewDec(9), pair.VirtualBase)
		require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

		ratio, found := k.GetRatio(ctx, "ukusd")
		require.NotNil(t, ratio.Ratio)
		require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *ratio.Ratio)

		require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))

		liq, found = k.GetLiquiditySum(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.NewInt(2), liq)

		pair, found = k.GetLiquidityPair(ctx, "ukusd")
		require.True(t, found)
		require.Equal(t, math.LegacyNewDec(19), pair.VirtualBase)
		require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

		ratio, found = k.GetRatio(ctx, "ukusd")
		require.NotNil(t, ratio.Ratio)
		require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *ratio.Ratio)

		require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))

		pair, found = k.GetLiquidityPair(ctx, "ukusd")
		require.Equal(t, math.LegacyNewDec(18), pair.VirtualBase)
		require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)

		require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))

		pair, found = k.GetLiquidityPair(ctx, "ukusd")
		require.Equal(t, math.LegacyNewDec(19), pair.VirtualBase)
		require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)

		return k.ExportGenesisBytes(ctx)
	})
}

func TestLiquidityPairs2(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)
	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1)
	require.NotNil(t, err)
}
