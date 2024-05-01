package keeper_test

import (
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestLiquidate1(t *testing.T) {
	k, dexMsg, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := mmMsg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
	})

	require.NoError(t, err)

	_, err = mmMsg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	})

	require.NoError(t, err)

	_, err = mmMsg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "24000",
	})

	require.NoError(t, err)

	_, err = dexMsg.Trade(ctx, &dextypes.MsgTrade{
		Creator:         keepertest.Alice,
		DenomFrom:       utils.BaseCurrency,
		DenomTo:         "ukusd",
		Amount:          "10000000",
		MaxPrice:        "",
		AllowIncomplete: true,
	})

	require.NoError(t, err)

	for i := 0; i < 10_000; i++ {
		k.ApplyInterest(ctx)
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	found, coin := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).Find("ukusd")
	require.True(t, found)
	vaultSize1 := coin.Amount

	loans := k.GetAllLoansByDenom(ctx, "ukusd")
	require.Equal(t, 1, len(loans))
	loanAmount1 := k.GetAllLoansByDenom(ctx, "ukusd")[0].Amount

	require.NoError(t, k.HandleLiquidations(ctx, ctx.EventManager()))

	found, coin = k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).Find("ukusd")
	require.True(t, found)
	vaultSize2 := coin.Amount
	require.True(t, vaultSize2.GT(vaultSize1))

	loanAmount2 := k.GetAllLoansByDenom(ctx, "ukusd")[0].Amount
	require.True(t, loanAmount2.LT(loanAmount1))
}

func TestLiquidate2(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := mmMsg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	})
	require.NoError(t, err)

	_, err = mmMsg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	})
	require.NoError(t, err)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	_, err = mmMsg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	})
	require.NoError(t, err)

	loanSize1, found := k.GetLoan(ctx, "ukusd", keepertest.Bob)
	require.True(t, found)

	k.ApplyInterest(ctx)
	require.NoError(t, k.HandleLiquidations(ctx, ctx.EventManager()))

	loanSize2, found := k.GetLoan(ctx, "ukusd", keepertest.Bob)
	require.True(t, found)

	require.True(t, loanSize2.Amount.LT(loanSize1.Amount))
}

func TestLiquidate3(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := mmMsg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	})
	require.NoError(t, err)

	_, err = mmMsg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	})
	require.NoError(t, err)

	collateralUser1, found1 := k.GetCollateral(ctx, "ukopi", keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	_, err = mmMsg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	})
	require.NoError(t, err)

	vaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vaultSize1 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf("ukusd")

	k.ApplyInterest(ctx)
	require.NoError(t, k.HandleLiquidations(ctx, ctx.EventManager()))

	vaultSize2 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf("ukusd")

	require.True(t, vaultSize2.GT(vaultSize1))

	collateralUser2, found2 := k.GetCollateral(ctx, "ukopi", keepertest.Bob)
	require.True(t, found2)

	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))
}

func TestLiquidate4(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := mmMsg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	})
	require.NoError(t, err)

	_, err = mmMsg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10000",
	})
	require.NoError(t, err)

	collateralUser1, found1 := k.GetCollateral(ctx, "ukusd", keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	_, err = mmMsg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	})
	require.NoError(t, err)

	k.ApplyInterest(ctx)
	require.NoError(t, k.HandleLiquidations(ctx, ctx.EventManager()))

	collateralUser2, found2 := k.GetCollateral(ctx, "ukusd", keepertest.Bob)
	require.True(t, found2)
	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))
}
