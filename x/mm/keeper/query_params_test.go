package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

func TestParamsQuery(t *testing.T) {
	_, keeper, ctx := keepertest.MmKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return keeper.SetParams(innerCtx, params)
	}))

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
