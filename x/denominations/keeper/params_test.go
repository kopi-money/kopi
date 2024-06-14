package keeper_test

import (
	"github.com/kopi-money/kopi/cache"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	params := types.DefaultParams()

	ctx = ctx.WithContext(cache.NewCacheContext(ctx.Context(), ctx.BlockHeight(), true))
	require.NoError(t, k.SetParams(ctx, params))

	require.Equal(t, len(params.DexDenoms), len(k.GetParams(ctx).DexDenoms))
	for i := 0; i < len(params.DexDenoms); i++ {
		require.EqualValues(t, params.DexDenoms[i], k.GetParams(ctx).DexDenoms[i])
	}

	require.EqualValues(t, params.DexDenoms, k.GetParams(ctx).DexDenoms)
	require.EqualValues(t, params.KCoins, k.GetParams(ctx).KCoins)
	require.EqualValues(t, params.CollateralDenoms, k.GetParams(ctx).CollateralDenoms)
	require.EqualValues(t, params.CAssets, k.GetParams(ctx).CAssets)

	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestSetParams(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	msg := denomkeeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)
	numDenoms1 := len(params.DexDenoms)

	_, err := msg.AddDEXDenom(ctx, &types.MsgAddDEXDenom{
		Authority:    k.GetAuthority(),
		Name:         "ukusd2",
		Factor:       "10",
		MinLiquidity: "1000",
		MinOrderSize: "1000",
	})
	require.NoError(t, err)

	params = k.GetParams(ctx)
	numDenoms2 := len(params.DexDenoms)
	require.Equal(t, numDenoms1+1, numDenoms2)

	_, err = msg.AddDEXDenom(ctx, &types.MsgAddDEXDenom{
		Authority:    k.GetAuthority(),
		Name:         "ukusd2",
		Factor:       "10",
		MinLiquidity: "1000",
		MinOrderSize: "1000",
	})
	require.Error(t, err)

	params = k.GetParams(ctx)
	numDenoms3 := len(params.DexDenoms)
	require.Equal(t, numDenoms2, numDenoms3)
}
