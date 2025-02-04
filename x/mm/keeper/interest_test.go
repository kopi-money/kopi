package keeper_test

import (
	"context"
	"testing"

	"github.com/kopi-money/kopi/constants"

	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestInterest1(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.BaseCurrency,
		Amount:  "20000000",
	}))

	require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "1000",
	}))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.ApplyInterest(innerCtx)
	})

	iterator := k.LoanIterator(ctx, constants.KUSD)
	require.Equal(t, 1, len(iterator.GetAll()))

	loanValue := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)
	require.True(t, loanValue.GT(math.LegacyNewDec(1000)))
}
