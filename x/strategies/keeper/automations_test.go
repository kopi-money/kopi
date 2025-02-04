package keeper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	dextypes "github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/strategies/keeper"
	"github.com/kopi-money/kopi/x/strategies/types"
	"github.com/stretchr/testify/require"
)

func TestAutomation1(t *testing.T) {
	_, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "1000"))

	require.Error(t, keepertest.AddAutomationMsg(ctx, msg, &types.MsgAutomationsAdd{}))
	require.Error(t, keepertest.AddAutomationMsg(ctx, msg, &types.MsgAutomationsAdd{Creator: keepertest.Alice, Title: "test"}))

	conditions := []types.Condition{
		{
			ConditionType: types.ConditionWalletAmount,
			String1:       constants.KUSD,
			Value:         math.LegacyNewDec(100),
			Comparison:    "GT",
		},
	}

	actions := []types.Action{
		{
			ActionType: types.ActionSendCoins,
			String1:    constants.BaseCurrency,
			String2:    keepertest.Bob,
			Amount:     "100%",
		},
	}

	rawConditions, err := json.Marshal(conditions)
	require.NoError(t, err)
	rawActions, err := json.Marshal(actions)
	require.NoError(t, err)

	require.NoError(t, keepertest.AddAutomationMsg(ctx, msg, &types.MsgAutomationsAdd{
		Creator:        keepertest.Alice,
		Title:          "title",
		IntervalType:   "1",
		IntervalLength: "1",
		ValidityType:   "1",
		ValidityValue:  "1",
		Conditions:     string(rawConditions),
		Actions:        string(rawActions),
	}))
}

func handleAutomation(ctx context.Context, k keeper.Keeper, automation types.Automation) (bool, []bool, error) {
	return handleAutomationAtHeight(ctx, k, automation)
}

func handleAutomationAtHeight(ctx context.Context, k keeper.Keeper, automation types.Automation) (bool, []bool, error) {
	p := k.GetParams(ctx)

	var totalConsumption uint64
	conditionsMatched, successfulActions, err := k.HandleAutomation(ctx, p, automation, 0, &totalConsumption)
	if err != nil {
		return conditionsMatched, successfulActions, err
	}

	if totalConsumption > 0 {
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, math.NewInt(int64(totalConsumption))))
			if err = k.BankKeeper.SendCoinsFromModuleToModule(innerCtx, types.PoolAutomationFunds, dextypes.PoolReserve, coins); err != nil {
				return fmt.Errorf("could not send funds from funds pool to reserve: %w", err)
			}

			return nil
		}); err != nil {
			return conditionsMatched, successfulActions, err
		}
	}

	return conditionsMatched, successfulActions, nil
}

func TestAutomation3(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "1000",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.False(t, allConditionsMatched)
	require.Nil(t, successfulActions)
}

func TestAutomation4(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "1"))

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "1000",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.False(t, allConditionsMatched)
	require.Nil(t, successfulActions)
}

func TestAutomation5(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	var (
		allConditionsMatched bool
		successfulActions    []bool
		err                  error
	)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "2"))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.KUSD).Amount.Int64()
	require.Equal(t, int64(2), poolBalance)
	require.Equal(t, int64(2), k.GetAutomationFunds(ctx, keepertest.Alice).Int64())

	allConditionsMatched, successfulActions, err = handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "1000",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.True(t, allConditionsMatched)
	require.True(t, successfulActions[0])

	poolBalance = k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.BaseCurrency).Amount.Int64()
	require.Equal(t, int64(0), poolBalance)
	require.Equal(t, int64(0), k.GetAutomationFunds(ctx, keepertest.Alice).Int64())
}

func TestAutomation6(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "3"))

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "LT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "1000",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.False(t, allConditionsMatched)
	require.Nil(t, successfulActions)

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.KUSD).Amount.Int64()
	require.Equal(t, int64(1), poolBalance)
	require.Equal(t, int64(1), k.GetAutomationFunds(ctx, keepertest.Alice).Int64())
}

func TestAutomation7(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "2"))

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "0",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.True(t, allConditionsMatched)
	require.False(t, successfulActions[0])

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.KUSD).Amount.Int64()
	require.Equal(t, int64(0), poolBalance)
	require.Equal(t, int64(0), k.GetAutomationFunds(ctx, keepertest.Alice).Int64())
}

func TestAutomation8(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	coins := sdk.NewCoins(
		sdk.NewCoin(constants.KUSD, math.NewInt(10000)),
		sdk.NewCoin(constants.BaseCurrency, math.NewInt(2)),
	)

	acc1, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	acc2, _ := sdk.AccAddressFromBech32(keepertest.Dave)
	require.NoError(t, k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc1, dextypes.PoolReserve, coins))
	require.NoError(t, k.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.PoolReserve, acc2, coins))

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Dave, "2"))

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Dave,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralAdd,
				String1:    constants.KUSD,
				Amount:     "-1",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.True(t, allConditionsMatched)
	require.True(t, successfulActions[0])

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.BaseCurrency).Amount.Int64()
	require.Equal(t, int64(0), poolBalance)
	require.Equal(t, int64(0), k.GetAutomationFunds(ctx, keepertest.Dave).Int64())

	userAcc, _ := sdk.AccAddressFromBech32(keepertest.Dave)
	userBalance := k.BankKeeper.SpendableCoin(ctx, userAcc, constants.KUSD).Amount.Int64()
	require.Equal(t, int64(1), userBalance)
}

func TestAutomation9(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "2"))

	allConditionsMatched, successfulActions, err := handleAutomation(ctx, k, types.Automation{
		Index:   0,
		Address: keepertest.Alice,
		Active:  true,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionCollateralWithdraw,
				String1:    constants.KUSD,
				Amount:     "1000",
			},
		},
		ValidityType: keeper.AutomationValidityUnlimited,
	})

	require.NoError(t, err)
	require.True(t, allConditionsMatched)
	require.False(t, successfulActions[0])

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), constants.KUSD).Amount.Int64()
	require.Equal(t, int64(0), poolBalance)
	require.Equal(t, int64(0), k.GetAutomationFunds(ctx, keepertest.Alice).Int64())
}

func TestAutomation10(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationValidityNumExecutions,
		ValidityValue: 2,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GTE",
				Value:         math.LegacyNewDec(100),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionSendCoins,
				String1:    constants.KUSD,
				String2:    keepertest.Bob,
				Amount:     "100",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, _, err := handleAutomation(ctx, k, automation)
	require.NoError(t, err)

	executedAutomation := k.GetAutomations(ctx)[0]
	require.Equal(t, int64(1), executedAutomation.TotalTimesExecuted)
	require.Equal(t, int64(1), executedAutomation.PeriodTimesExecuted)
	require.Equal(t, uint64(1), executedAutomation.TotalConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.TotalActionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodActionFeesConsumed)

	require.True(t, executedAutomation.Active)

	_, _, err = handleAutomation(ctx, k, *executedAutomation)
	require.NoError(t, err)

	_, _, err = handleAutomation(ctx, k, *executedAutomation)
	require.NoError(t, err)

	executedAutomation = k.GetAutomations(ctx)[0]
	require.False(t, executedAutomation.Active)
}

func TestAutomation11(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationValidityFeesConsumed,
		ValidityValue: 4,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GTE",
				Value:         math.LegacyNewDec(100),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionSendCoins,
				String1:    constants.KUSD,
				String2:    keepertest.Bob,
				Amount:     "100",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, _, err := handleAutomation(ctx, k, automation)
	require.NoError(t, err)

	executedAutomation := k.GetAutomations(ctx)[0]
	require.Equal(t, int64(1), executedAutomation.TotalTimesExecuted)
	require.Equal(t, int64(1), executedAutomation.PeriodTimesExecuted)
	require.Equal(t, uint64(1), executedAutomation.TotalConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.TotalActionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodActionFeesConsumed)

	require.True(t, executedAutomation.Active)

	_, _, err = handleAutomation(ctx, k, *executedAutomation)
	require.NoError(t, err)

	_, _, err = handleAutomation(ctx, k, *executedAutomation)
	require.NoError(t, err)

	executedAutomation = k.GetAutomations(ctx)[0]
	require.False(t, executedAutomation.Active)
}

func TestAutomation12(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationIntervalSeconds,
		ValidityValue: 10,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GTE",
				Value:         math.LegacyNewDec(100),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionSendCoins,
				String1:    constants.KUSD,
				String2:    keepertest.Bob,
				Amount:     "100",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, _, err := handleAutomationAtHeight(ctx, k, automation)
	require.NoError(t, err)

	executedAutomation := k.GetAutomations(ctx)[0]
	require.Equal(t, int64(1), executedAutomation.TotalTimesExecuted)
	require.Equal(t, int64(1), executedAutomation.PeriodTimesExecuted)
	require.Equal(t, uint64(1), executedAutomation.TotalConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodConditionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.TotalActionFeesConsumed)
	require.Equal(t, uint64(1), executedAutomation.PeriodActionFeesConsumed)

	require.True(t, executedAutomation.Active)

	_, _, err = handleAutomationAtHeight(ctx, k, *executedAutomation)
	require.NoError(t, err)

	executedAutomation = k.GetAutomations(ctx)[0]
	require.True(t, executedAutomation.Active)

	_, _, err = handleAutomationAtHeight(ctx, k, *executedAutomation)
	require.NoError(t, err)

	executedAutomation = k.GetAutomations(ctx)[0]
	require.True(t, executedAutomation.Active)
}

func TestAutomation13(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationIntervalSeconds,
		ValidityValue: 10,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       constants.KUSD,
				Comparison:    "GT",
				Value:         math.LegacyNewDec(100),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionSendCoins,
				String1:    keepertest.Carol,
				String2:    constants.KUSD,
				Amount:     "100%",
			},
			{
				ActionType: types.ActionSendCoins,
				String1:    keepertest.Carol,
				String2:    constants.KUSD,
				Amount:     "100",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, _, err := handleAutomation(ctx, k, automation)
	require.NoError(t, err)
}

func TestAutomation14(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationIntervalSeconds,
		ValidityValue: 10,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       "uwusdc",
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1_000_000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionDeposit,
				String1:    "uwusdc",
				Amount:     "100000000",
			},
			{
				ActionType: types.ActionCollateralAdd,
				String1:    "uwusdc",
				Amount:     "100000000",
			},
			{
				ActionType: types.ActionLoanBorrow,
				String1:    "uwusdc",
				Amount:     "1000000",
			},
			{
				ActionType: types.ActionLoanRepay,
				String1:    "uwusdc",
				Amount:     "1000000",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, executions, err := handleAutomation(ctx, k, automation)

	require.NoError(t, err)
	require.Equal(t, 4, countTrue(executions))
}

func TestAutomation15(t *testing.T) {
	k, msg, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.NoError(t, keepertest.AddAutomationFunds(ctx, msg, keepertest.Alice, "200"))

	automation := types.Automation{
		Index:         0,
		Address:       keepertest.Alice,
		Active:        true,
		ValidityType:  keeper.AutomationIntervalSeconds,
		ValidityValue: 10,
		Conditions: []types.Condition{
			{
				ConditionType: types.ConditionWalletAmount,
				String1:       "uwusdc",
				Comparison:    "GT",
				Value:         math.LegacyNewDec(1_000_000),
			},
		},
		Actions: []types.Action{
			{
				ActionType: types.ActionDeposit,
				String1:    "uwusdc",
				Amount:     "100000000",
			},
			{
				ActionType: types.ActionRedeem,
				String1:    "ucwusdc",
				Amount:     "100%",
			},
		},
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetAutomation(innerCtx, automation)
		return nil
	}))

	_, executions, err := handleAutomation(ctx, k, automation)

	require.NoError(t, err)
	require.Equal(t, 2, countTrue(executions))
}

func countTrue(executions []bool) (count int) {
	for _, execution := range executions {
		if execution {
			count++
		}
	}

	return
}
