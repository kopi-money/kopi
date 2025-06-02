package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestQueryOrderSum1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, _ := math.NewIntFromString("10000000000000000000")
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Alice,
		DenomGiving:    "inj",
		DenomReceiving: "ukusd",
		Amount:         "1000000",
		MaxPrice:       "50000000000",
		IsBuyOrder:     true,
	}))

	sum, err := k.CalcOrdersSum(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1_257538), sum.TruncateInt64())
}
