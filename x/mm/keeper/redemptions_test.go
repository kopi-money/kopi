package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestRedemptions1(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Carol,
		Denom:        "ukusd",
		CAssetAmount: "100",
		Fee:          "0",
	})

	require.Error(t, err)
}

func TestRedemptions2(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "10",
		Fee:          "0.1",
	}))

	require.NoError(t, handleRedemptions(ctx, k))

	iterator := k.RedemptionIterator(ctx, "ukusd")
	require.Equal(t, 0, len(iterator.GetAll()))

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance2 := coins.Find("ukusd")
	require.True(t, found)
	diff := balance1.Amount.Sub(balance2.Amount)
	require.Equal(t, math.NewInt(1), diff)
}

func TestRedemptions3(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "10000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Bob,
		Denom:   "ukopi",
		Amount:  "1000000",
	}))

	require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "9000",
	}))

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "2000",
		Fee:          "0.1",
	}))

	iterator := k.RedemptionIterator(ctx, "ukusd")
	redemptions := iterator.GetAll()
	require.Equal(t, 1, len(redemptions))

	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(2000), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)

	require.NoError(t, handleRedemptions(ctx, k))

	iterator = k.RedemptionIterator(ctx, "ukusd")
	redemptions = iterator.GetAll()
	require.Equal(t, 1, len(redemptions))

	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(1000), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)

	require.NoError(t, keepertest.PartiallyRepayLoan(ctx, msg, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "1",
	}))

	iterator = k.RedemptionIterator(ctx, "ukusd")
	redemptions = iterator.GetAll()
	require.Equal(t, 1, len(redemptions))

	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, int64(1000), redemptions[0].Amount.Int64())
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)
}

func TestRedemptions5(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "500",
	})

	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "500",
	})

	require.NoError(t, err)

	supply := k.BankKeeper.GetSupply(ctx, "uckusd").Amount
	require.Equal(t, math.NewInt(1000), supply)
}

func TestRedemptions6(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance1 := coins.Find("ukusd")
	require.True(t, found)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "1000",
	})

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "1000",
		Fee:          "0.5",
	}))

	require.NoError(t, handleRedemptions(ctx, k))

	iterator := k.RedemptionIterator(ctx, "ukusd")
	require.Equal(t, 0, len(iterator.GetAll()))

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance2 := coins.Find("ukusd")
	require.True(t, found)
	diff := balance1.Amount.Sub(balance2.Amount)
	require.Equal(t, math.NewInt(500), diff)
}

func TestRedemptions7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "1000",
	})

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "1000",
		Fee:          "0.5",
	}))

	iterator := k.RedemptionIterator(ctx, "ukusd")
	require.Equal(t, 1, len(iterator.GetAll()))
}
