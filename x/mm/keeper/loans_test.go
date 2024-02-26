package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoans1(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.Error(t, err)
}

func TestLoans2(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, err := msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.Error(t, err)
}

func TestLoans3(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	_, err := msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "101",
	})

	require.Error(t, err)
}

func TestLoans4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	_, err := msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	_, balance2 := coins.Find("ukusd")
	diff := balance2.Amount.Sub(balance1.Amount)
	require.Equal(t, math.NewInt(10), diff)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 1, len(loans))
	require.Equal(t, keepertest.Bob, loans[0].Address)
	require.Equal(t, math.LegacyNewDec(10), loans[0].Amount)
}

func TestLoans5(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	_, _ = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	_, err := msg.PartiallyRepayLoan(ctx, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "1",
	})

	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	_, balance2 := coins.Find("ukusd")
	diff := balance2.Amount.Sub(balance1.Amount)
	require.Equal(t, math.NewInt(9), diff)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 1, len(loans))
	require.Equal(t, keepertest.Bob, loans[0].Address)
	require.Equal(t, math.LegacyNewDec(9), loans[0].Amount)
}

func TestLoans6(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	_, _ = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	_, err := msg.PartiallyRepayLoan(ctx, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "11",
	})

	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	_, balance2 := coins.Find("ukusd")
	require.Equal(t, balance1.Amount, balance2.Amount)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 0, len(loans))
}

func TestLoans7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.NoError(t, err)

	_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	_, err = msg.RepayLoan(ctx, &types.MsgRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
	})

	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	_, balance2 := coins.Find("ukusd")
	require.Equal(t, balance1.Amount, balance2.Amount)

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 0, len(loans))
}

func TestLoans8(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})

	require.NoError(t, err)

	borrowable, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), borrowable)
}

func TestLoans9(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "10000",
	})

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)
	require.Equal(t, int64(500), withdrawable.Int64())
}

func TestLoans10(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "10000000",
	})

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(10000), withdrawable)
}

func TestLoans11(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100000000",
	})
	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "4050007",
	})
	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "4050006.884463958",
	})
	require.NoError(t, err)
}

func TestLoans12(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "10000000",
	})
	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})
	require.NoError(t, err)

	borrowable, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  borrowable.String(),
	})
	require.NoError(t, err)
}

func TestLoans13(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "10000000",
	})
	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})
	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	})
	require.NoError(t, err)

	borrowable1, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})
	require.NoError(t, err)

	borrowable2, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)

	borrowableInt1 := borrowable1.Int64()
	borrowableInt2 := borrowable2.Int64()

	require.Less(t, borrowableInt2, borrowableInt1)
}
