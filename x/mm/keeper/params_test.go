package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

func TestGetParams(t *testing.T) {
	_, k, ctx := keepertest.MmKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.SetParams(innerCtx, params)
	}))
	require.EqualValues(t, params, k.GetParams(ctx))
}
