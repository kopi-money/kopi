package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/stretchr/testify/require"
)

func TestLiquidityPairs1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	// Add 1 XKP
	// Expected:
	// Ratio: 0.1
	// Base:  1 + 0,  #entries: 1
	// Other: 0 + 0.1 #entries: 0
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))

	ratios := k.GetAllRatio(ctx)
	require.True(t, len(ratios) > 0)

	r, found := k.GetRatio(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *r.Ratio)

	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(0), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(0), liq.Int64())

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), pair.VirtualOther)

	// Add 1 kUSD
	// Expected:
	// Ratio: 0.1
	// Base:  1 + 9 #entries: 1
	// Other: 1 + 0 #entries: 1
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))

	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, math.NewInt(1), liq)

	pair, found = k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(9), pair.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	ratio, found := k.GetRatio(ctx, "ukusd")
	require.NotNil(t, ratio.Ratio)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *ratio.Ratio)

	// Add 1 kUSD
	// Expected:
	// Ratio: 0.1
	// Base:  1 + 19 #entries: 1
	// Other: 2 + 0  #entries: 2
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))

	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(2), liq.Int64())

	pair, found = k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(19), pair.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	ratio, found = k.GetRatio(ctx, "ukusd")
	require.NotNil(t, ratio.Ratio)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), *ratio.Ratio)

	// Add 1 XKP
	// Expected:
	// Ratio: 0.1
	// Base:  2 + 18 #entries: 2
	// Other: 2 + 0  #entries: 2
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))

	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(2), liq.Int64())

	pair, found = k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, math.LegacyNewDec(18), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)

	// Add 1 XKP
	// Expected:
	// Ratio: 0.1
	// Base:  1 + 19 #entries: 1
	// Other: 2 + 0  #entries: 2
	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1))

	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())

	pair, found = k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, math.LegacyNewDec(19), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)
}

func TestLiquidityPairs2(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)
	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1)
	require.NotNil(t, err)
}
