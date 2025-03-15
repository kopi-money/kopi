package keeper_test

import (
	"context"
	"cosmossdk.io/math"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestLiquidate1(t *testing.T) {
	k, dexMsg, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000_000_000)
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Alice, constants.KUSD, 1_000_000_000))

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100_000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.BaseCurrency,
		Amount:  "100_000000",
	}))

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, constants.KUSD)
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  availableToBorrow.String(),
	}))

	_, err = keepertest.Sell(ctx, dexMsg, &dextypes.MsgSell{
		Creator:        keepertest.Alice,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10_000_000000",
		MaxPrice:       "",
	})
	require.NoError(t, err)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		for i := 0; i < 10_000; i++ {
			if err = k.ApplyInterest(innerCtx); err != nil {
				return err
			}
		}

		return nil
	})

	iterator := k.LoanIterator(ctx, constants.KUSD)
	require.Equal(t, 1, len(iterator.GetAll()))

	loanValue1 := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)

	require.NoError(t, k.HandleLiquidations(ctx))

	loanValue2 := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)
	require.Less(t, loanValue2.TruncateInt().Int64(), loanValue1.TruncateInt().Int64())
}

func TestLiquidate2(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.BaseCurrency,
		Amount:  "1000000",
	}))

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, constants.KUSD)
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  availableToBorrow.String(),
	}))

	loanValue1 := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.ApplyInterest(innerCtx)
	}))

	require.NoError(t, k.HandleLiquidations(ctx))

	loanValue2 := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)
	require.True(t, loanValue2.LT(loanValue1))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestLiquidate3(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.BaseCurrency,
		Amount:  "1000000",
	}))

	userAcc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	balance1 := k.BankKeeper.SpendableCoins(ctx, userAcc)

	collateralUser1, found1 := k.LoadCollateral(ctx, constants.BaseCurrency, keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, constants.KUSD)
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  availableToBorrow.String(),
	}))

	vaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vaultSize1 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf(constants.KUSD)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.ApplyInterest(innerCtx)
	}))

	require.NoError(t, k.HandleLiquidations(ctx))

	balance2 := k.BankKeeper.SpendableCoins(ctx, userAcc)
	balanceDiff := balance2.AmountOf(constants.KUSD).Sub(balance1.AmountOf(constants.KUSD))

	vaultSize2 := k.BankKeeper.SpendableCoins(ctx, vaultAcc.GetAddress()).AmountOf(constants.KUSD)
	// When more collateral is sold than necessary, it is sent to the borrower. We add that amount to the vault to
	// test that collateral has been sold.
	vaultSize2 = vaultSize2.Add(balanceDiff)

	require.True(t, vaultSize2.GT(vaultSize1))

	collateralUser2, found2 := k.LoadCollateral(ctx, constants.BaseCurrency, keepertest.Bob)
	require.True(t, found2)

	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestLiquidate4(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "10000",
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	collateralUser1, found1 := k.LoadCollateral(ctx, constants.KUSD, keepertest.Bob)
	require.True(t, found1)

	availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Bob, constants.KUSD)
	require.NoError(t, err)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  availableToBorrow.String(),
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.ApplyInterest(innerCtx)
	}))

	require.NoError(t, k.HandleLiquidations(ctx))

	collateralUser2, found2 := k.LoadCollateral(ctx, constants.KUSD, keepertest.Bob)
	require.True(t, found2)
	require.True(t, collateralUser2.Amount.LT(collateralUser1.Amount))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestLiquidate5(t *testing.T) {
	k, dexMsg, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Alice, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Alice, constants.KUSD, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Alice, "uwusdc", 100_000000))

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "uwusdc",
		Amount:  "10_000000",
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "9_000000",
	}))

	r, _ := math.LegacyNewDecFromStr("0.26")

	setRatioKeeper, ok := k.DenomKeeper.(keepertest.SetRatioKeeper)
	require.True(t, ok)
	keepertest.SetRatio(ctx, setRatioKeeper, "uwusdc", r)

	require.NoError(t, k.HandleLiquidations(ctx))

	value := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)
	require.Equal(t, int64(8221154), value.TruncateInt().Int64())
}

func TestLiquidate6(t *testing.T) {
	k, _, mmMsg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "10_000000",
	}))

	require.NoError(t, checkCollateralSum(ctx, k))

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "9_000000",
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.ApplyInterest(innerCtx)
	}))

	require.NoError(t, k.HandleLiquidations(ctx))

	value := k.GetLoanValue(ctx, constants.KUSD, keepertest.Bob)
	require.Equal(t, int64(8550000), value.TruncateInt().Int64())
}
