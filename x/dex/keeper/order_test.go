package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/x/dex/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func createNOrder(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Order {
	items := make([]types.Order, n)
	for i := range items {
		items[i].Index = keeper.SetOrder(ctx, items[i])
	}
	return items
}

func TestOrderRemove(t *testing.T) {
	k, ctx, _ := keepertest.DexKeeper(t)
	items := createNOrder(&k, ctx, 10)
	for _, item := range items {
		k.RemoveOrder(ctx, item)
		_, found := k.GetOrder(ctx, item.Index)
		require.False(t, found)
	}
}
