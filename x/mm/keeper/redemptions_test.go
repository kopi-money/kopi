package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"testing"
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

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "10",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 0, len(redemptions))

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance2 := coins.Find("ukusd")
	require.True(t, found)
	diff := balance1.Amount.Sub(balance2.Amount)
	require.Equal(t, math.NewInt(1), diff)
}

func TestRedemptions3(t *testing.T) {
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
		Amount:  "1000",
	})

	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	_, err = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "10",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(10), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions = k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(10), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)

	_, err = msg.PartiallyRepayLoan(ctx, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "1",
	})

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions = k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(9), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)
}

func TestRedemptions4(t *testing.T) {
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
		Amount:  "1000",
	})

	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	_, err = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "10",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	_, err = msg.PartiallyRepayLoan(ctx, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "1",
	})

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(9), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)
}

func TestRedemptions5(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "5",
	})

	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "5",
	})

	require.NoError(t, err)

	supply := k.BankKeeper.GetSupply(ctx, "uckusd").Amount
	require.Equal(t, math.NewInt(10), supply)

	_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Carol,
		Denom:   "ukopi",
		Amount:  "1000",
	})

	require.NoError(t, err)

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Carol,
		Denom:   "ukusd",
		Amount:  "10",
	})

	require.NoError(t, err)

	_, err = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "5",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	_, err = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Bob,
		Denom:        "uckusd",
		CAssetAmount: "5",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	_, err = msg.PartiallyRepayLoan(ctx, &types.MsgPartiallyRepayLoan{
		Creator: keepertest.Carol,
		Denom:   "ukusd",
		Amount:  "5",
	})

	require.NoError(t, err)

	supply = k.BankKeeper.GetSupply(ctx, "uckusd").Amount
	require.Equal(t, supply, math.NewInt(10))

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(5), redemptions[0].Amount)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), redemptions[0].Fee)
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

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "1000",
		Fee:          "0.5",
	})

	require.NoError(t, err)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 0, len(redemptions))

	coins = k.BankKeeper.SpendableCoins(ctx, acc)
	found, balance2 := coins.Find("ukusd")
	require.True(t, found)
	diff := balance1.Amount.Sub(balance2.Amount)
	require.Equal(t, math.NewInt(500), diff)
}
