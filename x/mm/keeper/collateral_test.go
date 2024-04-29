package keeper_test

import (
	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCollateral1(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "kusd",
		Amount:  "100",
	})

	require.Error(t, err)
}

func TestCollateral2(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100",
	})

	require.NoError(t, err)

	collaterals := k.GetAllCollaterals(ctx, "ukopi")
	require.Equal(t, 1, len(collaterals))
}

func TestCollateral3(t *testing.T) {
	_, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "-100",
	})

	require.Error(t, err)
}

func TestCollateral4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukopi")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), withdrawable)
}

func TestCollateral5(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100",
	})

	_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukopi")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), withdrawable)

	withdrawable, err = k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), withdrawable)
}

func TestCollateral6(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100000",
	})

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
	})

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "1000",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukopi")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(80000), withdrawable)
}

func TestCollateral7(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	_, err := msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukopi",
		Amount:  "100000",
	})

	_, err = msg.AddCollateral(ctx, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
	})

	_, err = msg.AddDeposit(ctx, &types.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "100000",
	})

	_, err = msg.Borrow(ctx, &types.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   "ukusd",
		Amount:  "1000",
	})

	require.NoError(t, err)

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukopi")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100000), withdrawable)

	withdrawable, err = k.CalcWithdrawableCollateralAmount(ctx, keepertest.Alice, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100000), withdrawable)
}
