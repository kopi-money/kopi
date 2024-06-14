package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestLiquidate1(t *testing.T) {
	k, dexMsg, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "1000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "100000",
	}))

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	}))

	_, err = keepertest.Trade(ctx, dexMsg, &dextypes.MsgTrade{
		Creator:         keepertest.Alice,
		DenomFrom:       utils.BaseCurrency,
		DenomTo:         "ukusd",
		Amount:          "100000000",
		MaxPrice:        "",
		AllowIncomplete: true,
	})
	require.NoError(t, err)

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		for i := 0; i < 10_000; i++ {
			k.ApplyInterest(innerCtx)
		}

		return nil
	})

	iterator := k.LoanIterator(ctx, "ukusd")
	require.Equal(t, 1, len(iterator.GetAll()))

	loanValue1 := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.HandleLiquidations(innerCtx, innerCtx.EventManager())
	}))

	loanValue2 := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)
	require.True(t, loanValue2.LT(loanValue1))
}

func TestLiquidate2(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	}))

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	}))

	loanValue1 := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		k.ApplyInterest(innerCtx)
		return k.HandleLiquidations(innerCtx, innerCtx.EventManager())
	}))

	loanValue2 := k.GetLoanValue(ctx, "ukusd", keepertest.Bob)
	require.True(t, loanValue2.LT(loanValue1))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestLiquidate3(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	}))

	userAcc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	balance1 := k.BankKeeper.SpendableCoins(ctx, userAcc)

	collateralUser1, found1 := k.LoadCollateral(ctx, "ukopi", keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	}))

	vaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vaultSize1 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf("ukusd")

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		k.ApplyInterest(innerCtx)
		return k.HandleLiquidations(innerCtx, innerCtx.EventManager())
	}))

	balance2 := k.BankKeeper.SpendableCoins(ctx, userAcc)
	balanceDiff := balance2.AmountOf("ukusd").Sub(balance1.AmountOf("ukusd"))

	vaultSize2 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf("ukusd")
	// When more collateral is sold than necessary, it is sent to the borrower. We add that amount to the vault to
	// test that collateral has been sold.
	vaultSize2 = vaultSize2.Add(balanceDiff)

	require.True(t, vaultSize2.GT(vaultSize1))

	collateralUser2, found2 := k.LoadCollateral(ctx, "ukopi", keepertest.Bob)
	require.True(t, found2)

	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestLiquidate4(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10000",
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	collateralUser1, found1 := k.LoadCollateral(ctx, "ukusd", keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, "ukusd")
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  availableToBorrow.String(),
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		k.ApplyInterest(innerCtx)
		return k.HandleLiquidations(innerCtx, innerCtx.EventManager())
	}))

	collateralUser2, found2 := k.LoadCollateral(ctx, "ukusd", keepertest.Bob)
	require.True(t, found2)
	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))

	require.NoError(t, checkCollateralSum(ctx, k))
}
