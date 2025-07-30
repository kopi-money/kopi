package keeper_test

import (
	"context"
	tokenfactorykeeper "github.com/kopi-money/kopi/x/tokenfactory/keeper"
	"testing"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/constants"
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
	}), dextypes.ErrNotEnoughFunds)

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

func TestActions3(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	factoryK := k.FactoryKeeper.(tokenfactorykeeper.Keeper)
	factoryMsgServer := tokenfactorykeeper.NewMsgServerImpl(factoryK)

	fullName, err := keepertest.CreateFactoryDenom(ctx, factoryMsgServer, accAddress.String(), "test", "test", 6)
	require.NoError(t, err)

	_, has := factoryK.GetDenomByFullName(ctx, fullName)
	require.True(t, has)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, factoryMsgServer, keepertest.Alice, fullName, keepertest.Alice, "1000"))
	require.NoError(t, keepertest.CreatePool(ctx, factoryMsgServer, keepertest.Alice, fullName, "1000", "ukusd", "1000", "0.01", 300))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    constants.BaseCurrency,
		String2:    keepertest.Bob,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    constants.BaseCurrency,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    constants.BaseCurrency,
		String2:    keepertest.Bob,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		String2:    constants.BaseCurrency,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddBoth,
		String1:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddBoth,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddBoth,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddDexDenom,
		String1:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddDexDenom,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddDexDenom,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddFactoryDenom,
		String1:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddFactoryDenom,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityAddFactoryDenom,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.Error(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityWithdraw,
		String1:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityWithdraw,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, k.CheckAction(ctx, keepertest.Alice, types.Action{
		ActionType: types.ActionFactoryLiquidityWithdraw,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))
}

func TestActions4(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	factoryK := k.FactoryKeeper.(tokenfactorykeeper.Keeper)
	factoryMsgServer := tokenfactorykeeper.NewMsgServerImpl(factoryK)

	fullName, err := keepertest.CreateFactoryDenom(ctx, factoryMsgServer, accAddress.String(), "test", "test", 6)
	require.NoError(t, err)

	_, has := factoryK.GetDenomByFullName(ctx, fullName)
	require.True(t, has)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, factoryMsgServer, keepertest.Alice, fullName, keepertest.Alice, "10000"))
	require.NoError(t, keepertest.CreatePool(ctx, factoryMsgServer, keepertest.Alice, fullName, "10000", "ukusd", "10000", "0.01", 300))

	require.ErrorContains(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "1",
	}), "trade amount too small")

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "2000",
	}))

	require.NoError(t, keepertest.MintFactoryDenom(ctx, factoryMsgServer, keepertest.Alice, fullName, keepertest.Alice, "10000"))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactorySell,
		String1:    fullName,
		String2:    fullName,
		Amount:     "2000",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "2000",
	}))

	require.NoError(t, keepertest.MintFactoryDenom(ctx, factoryMsgServer, keepertest.Alice, fullName, keepertest.Alice, "10000"))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryBuy,
		String1:    fullName,
		String2:    fullName,
		Amount:     "2000",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddBoth,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddBoth,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddDexDenom,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddDexDenom,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddFactoryDenom,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityAddFactoryDenom,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityWithdraw,
		String1:    fullName,
		String2:    fullName,
		Amount:     "100",
	}))

	require.NoError(t, executeAction(ctx, k, accAddress, types.Action{
		ActionType: types.ActionFactoryLiquidityWithdraw,
		String1:    fullName,
		String2:    "ukusd",
		Amount:     "100",
	}))

	//ActionFactoryLiquidityAddBoth
	//ActionFactoryLiquidityAddDexDenom
	//ActionFactoryLiquidityAddFactoryDenom
	//ActionFactoryLiquidityWithdraw
}
