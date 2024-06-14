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

	// Add 8 XKP
	// Expected:
	// Ratio: 0.25
	// Base:  8 + 0,   #entries: 1
	// Other: 0 + 2 #entries: 0
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 8))

	ratios := k.GetAllRatio(ctx)
	require.Equal(t, 0, len(ratios))

	r, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), r.Ratio)

	require.Equal(t, int64(8), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(0), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(8), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(0), liq.Int64())

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(2), pair.VirtualOther)

	// Add 1 kUSD
	// Expected:
	// Ratio: 0.25
	// Base:  8 + 0 #entries: 1
	// Other: 1 + 1 #entries: 1
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))

	require.Equal(t, int64(8), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(1), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(8), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(1), liq.Int64())

	pair, err = k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(1), pair.VirtualOther)

	ratio, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.NotNil(t, ratio.Ratio)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), ratio.Ratio)

	// Add 1 kUSD
	// Expected:
	// Ratio: 0.25
	// Base:  8 + 0 #entries: 1
	// Other: 2 + 0  #entries: 2
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1))

	require.Equal(t, int64(8), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(8), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(2), liq.Int64())

	pair, err = k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, int64(0), pair.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(0), pair.VirtualOther.TruncateInt().Int64())

	ratio, err = k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.NotNil(t, ratio.Ratio)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), ratio.Ratio)

	// Add 4 XKP
	// Expected:
	// Ratio: 0.25
	// Base:  12 + 0 #entries: 2
	// Other: 2  + 1 #entries: 2
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 4))

	require.Equal(t, int64(12), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(12), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(2), liq.Int64())

	pair, err = k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(1), pair.VirtualOther)

	// Remove 4 XKP
	// Expected:
	// Ratio: 0.25
	// Base:  8 + 0 #entries: 1
	// Other: 2 + 0 #entries: 2
	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 4))

	require.Equal(t, int64(8), k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice).Int64())
	require.Equal(t, int64(2), k.GetLiquidityByAddress(ctx, "ukusd", keepertest.Alice).Int64())

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(8), liq.Int64())
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, int64(2), liq.Int64())

	pair, err = k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, int64(0), pair.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(0), pair.VirtualOther.TruncateInt().Int64())
}

func TestLiquidityPairs2(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)
	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1)
	require.NotNil(t, err)
}
