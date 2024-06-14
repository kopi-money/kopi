package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestInterest1(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "20000000",
	}))

	require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "1000",
	}))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		k.ApplyInterest(innerCtx)
		return nil
	})

	iterator := k.LoanIterator(ctx, "ukusd")
	require.Equal(t, 1, len(iterator.GetAll()))

	loanValue := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)
	require.True(t, loanValue.GT(math.LegacyNewDec(1000)))
}
