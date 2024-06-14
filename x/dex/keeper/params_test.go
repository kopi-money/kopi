package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _ := keepertest.DexKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.SetParams(innerCtx, params)
	}))
	require.EqualValues(t, params, k.GetParams(ctx))
}
