package keeper_test

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	"github.com/kopi-money/kopi/constants"

	"cosmossdk.io/math"
	mmkeeper "github.com/kopi-money/kopi/x/mm/keeper"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestCollateral1(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "kusd",
		Amount:  "100",
	})

	require.Error(t, err)

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral2(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	}))

	iterator := k.CollateralIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral3(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.Error(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "-100",
	}))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	}))

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, int64(100), withdrawable.TruncateInt().Int64())

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral5(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, int64(100), withdrawable.TruncateInt().Int64())

	withdrawable, err = k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, int64(100), withdrawable.TruncateInt().Int64())

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral6(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "1000",
	}))

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, int64(92000), withdrawable.TruncateInt().Int64())

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100000",
	}))

	require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "1000",
	}))

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, int64(100000), withdrawable.TruncateInt().Int64())

	withdrawable, err = k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, int64(100000), withdrawable.TruncateInt().Int64())

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral8(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100000",
	}))

	require.NoError(t, k.HandleRedemptions(ctx))

	require.NoError(t, checkCollateralSum(ctx, k))
}

func TestCollateral9(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	accAlice, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	accBob, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	balanceAlice1 := k.BankKeeper.SpendableCoin(ctx, accAlice, constants.BaseCurrency)
	balanceBob1 := k.BankKeeper.SpendableCoin(ctx, accBob, constants.BaseCurrency)

	require.NoError(t, keepertest.AddCollateralForBeneficiary(ctx, msg, &types.MsgAddCollateralForBeneficiary{
		Creator:     keepertest.Alice,
		Beneficiary: keepertest.Bob,
		Denom:       constants.BaseCurrency,
		Amount:      "100000",
	}))

	balanceAlice2 := k.BankKeeper.SpendableCoin(ctx, accAlice, constants.BaseCurrency)
	balanceBob2 := k.BankKeeper.SpendableCoin(ctx, accBob, constants.BaseCurrency)

	require.True(t, balanceAlice1.IsGTE(balanceAlice2))
	require.True(t, balanceBob1.Equal(balanceBob2))

	iterator := k.CollateralIterator(ctx, constants.BaseCurrency)
	collaterals := iterator.GetAll()
	require.Equal(t, 1, len(collaterals))

	collateral := collaterals[0]
	require.NotEqual(t, collateral.Address, keepertest.Alice)
	require.Equal(t, collateral.Address, keepertest.Bob)

	require.NoError(t, checkCollateralSum(ctx, k))
}

func checkCollateralSum(ctx context.Context, k mmkeeper.Keeper) error {
	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral)

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		poolAmount := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress()).AmountOf(denom.DexDenom)
		collateralSum := getCollateralSum(ctx, k, denom.DexDenom)

		if !poolAmount.Equal(collateralSum) {
			return fmt.Errorf("amount for %v don't match: %v vs %v", denom.DexDenom, poolAmount.String(), collateralSum.String())
		}
	}

	return nil
}

func getCollateralSum(ctx context.Context, k mmkeeper.Keeper, denom string) math.Int {
	sum := math.ZeroInt()
	iterator := k.CollateralIterator(ctx, denom)

	for _, entry := range iterator.GetAll() {
		sum = sum.Add(entry.Amount)
	}

	return sum
}
