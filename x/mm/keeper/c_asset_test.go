package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestDeposit1(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100",
	})

	require.Error(t, err)
}

func TestDeposit2(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.NoError(t, err)

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount, math.NewInt(100))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	found, coin := k.BankKeeper.SpendableCoins(ctx, acc).Find("uckusd")
	require.True(t, found)
	require.Equal(t, coin.Amount, math.NewInt(100))
}

func TestDeposit3(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(100), redemptions[0].Amount)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions = k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 0, len(redemptions))

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount, math.NewInt(0))
}

func TestDeposit4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "50",
		Fee:          "0.1",
	})

	require.NoError(t, err)

	redemptions := k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(50), redemptions[0].Amount)

	require.NoError(t, k.HandleRedemptions(ctx, ctx.EventManager()))

	redemptions = k.GetRedemptions(ctx, "uckusd")
	require.Equal(t, 0, len(redemptions))

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount, math.NewInt(50))
}

func TestDeposit5(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, err := msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "ukusd",
		CAssetAmount: "200",
		Fee:          "0.1",
	})

	require.Error(t, err)
}

func TestDeposit6(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.CancelRedemptionRequest(ctx, &types.MsgCancelRedemptionRequest{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
	})

	require.Error(t, err)
}

func TestDeposit7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.1",
	})

	_, err := msg.CancelRedemptionRequest(ctx, &types.MsgCancelRedemptionRequest{
		Creator: keepertest.Alice,
		Denom:   "uckusd",
	})

	require.NoError(t, err)

	redemptions := k.GetRedemptions(ctx, "ukusd")
	require.Equal(t, 0, len(redemptions))
}

func TestDeposit8(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, err := msg.UpdateRedemptionRequest(ctx, &types.MsgUpdateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "ukusd",
		Fee:          "0",
		CAssetAmount: "50",
	})

	require.Error(t, err)
}

func TestDeposit9(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	_, _ = msg.CreateRedemptionRequest(ctx, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.1",
	})

	_, err := msg.UpdateRedemptionRequest(ctx, &types.MsgUpdateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		Fee:          "0.5",
		CAssetAmount: "50",
	})

	require.NoError(t, err)

	redReq, found := k.GetRedemption(ctx, "uckusd", keepertest.Alice)
	require.True(t, found)
	require.Equal(t, redReq.Fee, math.LegacyNewDecWithPrec(5, 1))
	require.Equal(t, redReq.Amount, math.NewInt(50))
}

func TestDeposit10(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.NoError(t, err)

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Bob,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.NoError(t, err)

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount, math.NewInt(200))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	found, coin := k.BankKeeper.SpendableCoins(ctx, acc).Find("uckusd")
	require.True(t, found)
	require.Equal(t, coin.Amount, math.NewInt(100))

	acc, _ = sdk.AccAddressFromBech32(keepertest.Bob)
	found, coin = k.BankKeeper.SpendableCoins(ctx, acc).Find("uckusd")
	require.True(t, found)
	require.Equal(t, coin.Amount, math.NewInt(100))
}

func TestDeposit11(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)
	addr, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	depositAmount := 1_000

	for range 10 {
		_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
			Creator: keepertest.Alice,
			Denom:   "ukusd",
			Amount:  strconv.Itoa(depositAmount),
		})
		require.NoError(t, err)

		cAssetSupply := k.BankKeeper.SpendableCoin(ctx, addr, "uckusd")
		require.Greater(t, cAssetSupply.Amount.Int64(), int64(0))

		_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
			Creator: keepertest.Alice,
			Denom:   "uckusd",
			Amount:  cAssetSupply.Amount.String(),
		})
		require.NoError(t, err)

		var availableToBorrow math.Int
		availableToBorrow, err = k.CalcAvailableToBorrow(ctx, keepertest.Alice, "ukusd")
		require.NoError(t, err)
		require.Greater(t, availableToBorrow.Int64(), int64(0))

		_, err = msg.Borrow(ctx, &types.MsgBorrow{
			Creator: keepertest.Alice,
			Denom:   "ukusd",
			Amount:  availableToBorrow.String(),
		})
		require.NoError(t, err)

		require.Less(t, int(availableToBorrow.Int64()), depositAmount)
		depositAmount = int(availableToBorrow.Int64())
	}

}
