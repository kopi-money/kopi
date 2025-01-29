package keeper_test

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
	"testing"
)

type SetCollateralLTV interface {
	CollateralUpdateLTV(ctx context.Context, denom, ltvString string) error
}

func TestDelistCollateral1(t *testing.T) {
	k, _, _, ctx := keepertest.SetupMMMsgServer(t)
	require.Error(t, k.DelistCollateralDenom(ctx, constants.BaseCurrency))
}

func TestDelistCollateral2(t *testing.T) {
	k, _, _, ctx := keepertest.SetupMMMsgServer(t)
	require.Error(t, k.DelistCollateralDenom(ctx, "blabla"))
}

func TestDelistCollateral3(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	}))

	setCollateral, ok := k.DenomKeeper.(SetCollateralLTV)
	require.True(t, ok)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return setCollateral.CollateralUpdateLTV(innerCtx, constants.BaseCurrency, "0.1")
	}))

	require.Error(t, k.DelistCollateralDenom(ctx, "blabla"))
}

func TestDelistCollateral4(t *testing.T) {
	k, _, msg, ctx := keepertest.SetupMMMsgServer(t)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	balance1 := k.BankKeeper.SpendableCoin(ctx, acc, constants.BaseCurrency).Amount

	require.NoError(t, keepertest.AddCollateral(ctx, msg, &types.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "100",
	}))

	balance2 := k.BankKeeper.SpendableCoin(ctx, acc, constants.BaseCurrency).Amount
	require.NotEqual(t, balance1.Int64(), balance2.Int64())

	setCollateral, ok := k.DenomKeeper.(SetCollateralLTV)
	require.True(t, ok)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return setCollateral.CollateralUpdateLTV(innerCtx, constants.BaseCurrency, "0")
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.DelistCollateralDenom(innerCtx, constants.BaseCurrency)
	}))

	balance3 := k.BankKeeper.SpendableCoin(ctx, acc, constants.BaseCurrency).Amount
	require.Equal(t, balance1.Int64(), balance3.Int64())
}
