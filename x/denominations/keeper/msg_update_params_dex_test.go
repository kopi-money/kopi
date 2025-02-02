package keeper_test

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/constants"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	"github.com/kopi-money/kopi/x/denominations/types"
	"github.com/stretchr/testify/require"
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
)

func TestUpdate1(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)

	minLiq1 := k.MinLiquidity(ctx, constants.BaseCurrency)
	require.True(t, updateWithPanic(ctx, k, "1"))
	minLiq2 := k.MinLiquidity(ctx, constants.BaseCurrency)

	require.Equal(t, minLiq1.Int64(), minLiq2.Int64())
}

func updateWithPanic(ctx context.Context, k denomkeeper.Keeper, minLiq string) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()

	_ = k.DexUpdateMinimumLiquidity(ctx, constants.BaseCurrency, minLiq)
	return
}

func TestUpdate2(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)

	minLiq1 := k.MinLiquidity(ctx, constants.BaseCurrency)

	var panicked bool
	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.DexUpdateMinimumLiquidity(innerCtx, constants.BaseCurrency, "-1")
	}))

	require.False(t, panicked)

	minLiq2 := k.MinLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, minLiq1.Int64(), minLiq2.Int64())
}

func TestUpdate3(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)

	minLiq1 := k.MinLiquidity(ctx, constants.BaseCurrency)

	var panicked bool
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.DexUpdateMinimumLiquidity(innerCtx, constants.BaseCurrency, "1")
	}))

	require.False(t, panicked)

	minLiq2 := k.MinLiquidity(ctx, constants.BaseCurrency)
	require.NotEqual(t, minLiq1.Int64(), minLiq2.Int64())
}

func TestUpdate4(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.SetParams(innerCtx, types.DefaultParams())
	}))

	minLiq1 := k.MinLiquidity(ctx, constants.BaseCurrency)

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		params := k.GetParams(innerCtx)

		dexDenom := params.DexDenoms[0]
		dexDenom.MinLiquidity = math.NewInt(-1)

		params.DexDenoms = []types.DexDenom{
			dexDenom,
		}

		return k.SetParams(innerCtx, params)
	}))

	minLiq2 := k.MinLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, minLiq1.Int64(), minLiq2.Int64())
}
