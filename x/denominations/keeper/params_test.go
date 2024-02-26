package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	params := types.DefaultParams()

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
