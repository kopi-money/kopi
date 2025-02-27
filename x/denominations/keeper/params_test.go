package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"

	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		require.NoError(t, k.SetParams(innerCtx, params))

		require.Equal(t, len(params.DexDenoms), len(k.GetParams(innerCtx).DexDenoms))
		for i := 0; i < len(params.DexDenoms); i++ {
			require.EqualValues(t, params.DexDenoms[i], k.GetParams(innerCtx).DexDenoms[i])
		}

		require.EqualValues(t, params.DexDenoms, k.GetParams(innerCtx).DexDenoms)
		require.EqualValues(t, params.KCoins, k.GetParams(innerCtx).KCoins)
		require.EqualValues(t, params.CollateralDenoms, k.GetParams(innerCtx).CollateralDenoms)
		require.EqualValues(t, params.CAssets, k.GetParams(innerCtx).CAssets)

		require.EqualValues(t, params, k.GetParams(innerCtx))

		return nil
	}))
}

func TestSetParams(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	msg := denomkeeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)
	numDenoms1 := len(params.DexDenoms)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msg.DexAddDenom(innerCtx, &types.MsgDexAddDenom{
			Authority:    k.GetAuthority(),
			Name:         "ukusd2",
			Factor:       "10",
			MinLiquidity: "1000",
			MinOrderSize: "1000",
			Exponent:     6,
		})
		return err
	}))

	params = k.GetParams(ctx)
	numDenoms2 := len(params.DexDenoms)
	require.Equal(t, numDenoms1+1, numDenoms2)

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msg.DexAddDenom(innerCtx, &types.MsgDexAddDenom{
			Authority:    k.GetAuthority(),
			Name:         "ukusd2",
			Factor:       "10",
			MinLiquidity: "1000",
			MinOrderSize: "1000",
		})
		return err
	}))

	params = k.GetParams(ctx)
	numDenoms3 := len(params.DexDenoms)
	require.Equal(t, numDenoms2, numDenoms3)
}
