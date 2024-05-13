package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestInterest1(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
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
		Amount:  "1000",
	})

	require.NoError(t, err)

	k.ApplyInterest(ctx)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 1, len(loans))

	loanValue := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)
	require.True(t, loanValue.GT(math.LegacyNewDec(1000)))
}
