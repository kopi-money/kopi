package keeper_test

import (
	"context"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/strategies/keeper"
	"github.com/kopi-money/kopi/x/strategies/types"
	"github.com/stretchr/testify/require"
)

func TestActions1(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{}))
	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionSell,
		String1:    constants.BaseCurrency,
		String2:    constants.KUSD,
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionBuy,
		String1:    constants.BaseCurrency,
		String2:    constants.KUSD,
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionSell,
		String1:    constants.BaseCurrency,
		String2:    constants.BaseCurrency,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionSell,
		String1:    constants.BaseCurrency,
		String2:    constants.KUSD,
		Amount:     "0%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionDeposit,
		String1:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionDeposit,
		String1:    constants.BaseCurrency,
		String2:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionRedeem,
		String1:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionRedeem,
		String1:    constants.BaseCurrency,
		String2:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionCollateralAdd,
		String1:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionCollateralAdd,
		String1:    constants.BaseCurrency,
		String2:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionCollateralWithdraw,
		String1:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionCollateralWithdraw,
		String1:    constants.BaseCurrency,
		String2:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionSendCoins,
		String1:    keepertest.Alice,
		String2:    constants.BaseCurrency,
		Amount:     "100%",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionSendCoins,
		String1:    constants.BaseCurrency,
		String2:    keepertest.Bob,
		Amount:     "100%",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionStake,
		String1:    "random",
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionDepositAutomationFunds,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionWithdrawAutomationFunds,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionRedeem,
		String1:    "uckusd",
		Amount:     "0",
	}))
}

func TestActions2(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)
	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionDeposit,
		String1:    constants.KUSD,
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionDeposit,
		String1:    "unknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidDexAsset)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionRedeem,
		String1:    "uckusd",
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionRedeem,
		String1:    "unknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidDexAsset)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionCollateralAdd,
		String1:    constants.KUSD,
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionCollateralAdd,
		String1:    "uknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidDexAsset)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionCollateralWithdraw,
		String1:    constants.KUSD,
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionCollateralWithdraw,
		String1:    "unknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidCollateralDenom)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionLiquidityAdd,
		String1:    constants.BaseCurrency,
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionLiquidityAdd,
		String1:    "uknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidDexAsset)

	require.NoError(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionLiquidityWithdraw,
		String1:    constants.BaseCurrency,
		String2:    "",
		Amount:     "1000",
	}))

	require.ErrorIs(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionLiquidityWithdraw,
		String1:    "uknown",
		String2:    "",
		Amount:     "1000",
	}), denomtypes.ErrInvalidDexAsset)

	require.Error(t, executeAction(ctx, k, acc, types.Action{
		ActionType: types.ActionLoanBorrow,
		String1:    constants.KUSD,
		String2:    "",
		Amount:     "1",
	}))
}

func executeAction(ctx context.Context, k keeper.Keeper, acc sdk.AccAddress, action types.Action) error {
	return cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		return k.ExecuteAction(innerCtx, acc, action, 0, 0, 0)
	})
}
