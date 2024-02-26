package keeper_test

import (
	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInterest1(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "20000000",
	})

	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	k.ApplyInterest(ctx)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 1, len(loans))
	require.True(t, loans[0].Amount.GT(math.LegacyNewDec(10)))
}
