package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestOrdersRemove1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "1",
	}))

	require.Equal(t, 1, k.GetAllOrdersNum())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RemoveDenomOrders(innerCtx, "uwusdc")
	}))

	require.Equal(t, 1, k.GetAllOrdersNum())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RemoveDenomOrders(innerCtx, constants.KUSD)
	}))

	require.Equal(t, 0, k.GetAllOrdersNum())
}

func TestOrdersRemove2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "1",
	}))

	require.Equal(t, 1, k.GetAllOrdersNum())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RemoveDenomOrders(innerCtx, constants.BaseCurrency)
	}))

	require.Equal(t, 0, k.GetAllOrdersNum())
}
