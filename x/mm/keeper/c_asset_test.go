package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/constants"

	"github.com/cosmos/cosmos-sdk/cache"
	mmkeeper "github.com/kopi-money/kopi/x/mm/keeper"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func TestDeposit1(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	})

	require.Error(t, err)
}

func TestDeposit2(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
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

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.05",
	}))

	iterator := k.RedemptionIterator(ctx, constants.KUSD)
	redemptions := iterator.GetAll()
	require.Equal(t, 1, len(redemptions))
	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(100), redemptions[0].Amount)

	require.NoError(t, handleRedemptions(ctx, k))

	iterator = k.RedemptionIterator(ctx, constants.KUSD)
	require.Equal(t, 0, len(iterator.GetAll()))

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount, math.NewInt(0))
}

func handleRedemptions(ctx context.Context, k mmkeeper.Keeper) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.HandleRedemptions(innerCtx)
	})
}

func TestDeposit4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, _ = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	})

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "50",
		Fee:          "0.05",
	}))

	iterator := k.RedemptionIterator(ctx, constants.KUSD)
	redemptions := iterator.GetAll()
	require.Equal(t, 1, len(redemptions))

	require.Equal(t, keepertest.Alice, redemptions[0].Address)
	require.Equal(t, math.NewInt(50), redemptions[0].Amount)

	require.NoError(t, handleRedemptions(ctx, k))

	iterator = k.RedemptionIterator(ctx, constants.KUSD)
	require.Equal(t, 0, len(iterator.GetAll()))

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, int64(50), supply.Amount.Int64())
}

func TestDeposit5(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	require.Error(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        constants.KUSD,
		CAssetAmount: "200",
		Fee:          "0.1",
	}))
}

func TestDeposit6(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.CancelRedemptionRequest(ctx, &types.MsgCancelRedemptionRequest{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
	})

	require.Error(t, err)
}

func TestDeposit7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.05",
	}))

	require.NoError(t, keepertest.CancelRedemptionRequest(ctx, msg, &types.MsgCancelRedemptionRequest{
		Creator: keepertest.Alice,
		Denom:   "uckusd",
	}))

	iterator := k.RedemptionIterator(ctx, constants.KUSD)
	require.Equal(t, 0, len(iterator.GetAll()))
}

func TestDeposit8(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	_, err := msg.UpdateRedemptionRequest(ctx, &types.MsgUpdateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        constants.KUSD,
		Fee:          "0",
		CAssetAmount: "50",
	})

	require.Error(t, err)
}

func TestDeposit9(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	require.NoError(t, keepertest.CreateRedemptionRequest(ctx, msg, &types.MsgCreateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		CAssetAmount: "100",
		Fee:          "0.05",
	}))

	require.NoError(t, keepertest.UpdateRedemptionRequest(ctx, msg, &types.MsgUpdateRedemptionRequest{
		Creator:      keepertest.Alice,
		Denom:        "uckusd",
		Fee:          "0.04",
		CAssetAmount: "50",
	}))

	redReq, found := k.LoadRedemptionRequest(ctx, constants.KUSD, keepertest.Alice)
	require.True(t, found)
	require.Equal(t, redReq.Fee, math.LegacyNewDecWithPrec(4, 2))
	require.Equal(t, redReq.Amount, math.NewInt(50))
}

func TestDeposit10(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
		Creator: keepertest.Bob,
		Denom:   constants.KUSD,
		Amount:  "100",
	}))

	supply := k.BankKeeper.GetSupply(ctx, "uckusd")
	require.Equal(t, supply.Amount.Int64(), int64(200))

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

	depositAmount := 100_000

	for range 10 {
		require.NoError(t, keepertest.AddDeposit(ctx, msg, &types.MsgAddDeposit{
			Creator: keepertest.Alice,
			Denom:   constants.KUSD,
			Amount:  strconv.Itoa(depositAmount),
		}))

		cAssetSupply := k.BankKeeper.SpendableCoin(ctx, addr, "uckusd")
		require.Greater(t, cAssetSupply.Amount.Int64(), int64(0))

		require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
			Creator: keepertest.Alice,
			Denom:   "uckusd",
			Amount:  cAssetSupply.Amount.String(),
		}))

		var availableToBorrow math.Int
		availableToBorrow, err := k.CalcAvailableToBorrow(ctx, keepertest.Alice, constants.KUSD)
		require.NoError(t, err)
		require.Greater(t, availableToBorrow.Int64(), int64(0))

		require.NoError(t, keepertest.Borrow(ctx, msg, &types.MsgBorrow{
			Creator: keepertest.Alice,
			Denom:   constants.KUSD,
			Amount:  availableToBorrow.String(),
		}))

		require.Less(t, int(availableToBorrow.Int64()), depositAmount)
		depositAmount = int(availableToBorrow.Int64())
	}

}
