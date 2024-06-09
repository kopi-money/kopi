package keeper_test

import (
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/x/dex/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/testutil/nullify"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNRatio(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Ratio {
	items := make([]types.Ratio, n)
	for i := range items {
		items[i].Denom = strconv.Itoa(i)

		keeper.SetRatio(ctx, items[i])
	}
	return items
}

func TestRatioGet(t *testing.T) {
	keeper, _, ctx := keepertest.SetupDexMsgServer(t)
	items := createNRatio(&keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetRatio(ctx,
			item.Denom,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
