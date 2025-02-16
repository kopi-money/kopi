package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
	"github.com/kopi-money/kopi/x/strategies/types"
	"github.com/stretchr/testify/require"
)

func TestConditions1(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.Error(t, k.ValidateCondition(ctx, types.Condition{}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionPrice,
		String1:       constants.KUSD,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(25, 2),
		Comparison:    "",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionPrice,
		String1:       constants.BaseCurrency,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(25, 2),
		Comparison:    "GT",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionPrice,
		String1:       constants.BaseCurrency,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyDec{},
		Comparison:    "GT",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionPrice,
		String1:       constants.KUSD,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(25, 2),
		Comparison:    "GT",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GT",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GT",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.KUSD,
		String2:       constants.KUSD,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "E",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.KUSD,
		String2:       constants.KUSD,
		Value:         math.LegacyNewDec(0),
		Comparison:    "GT",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.KUSD,
		String2:       constants.KUSD,
		Value:         math.LegacyNewDec(-1),
		Comparison:    "GT",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionWalletValue,
		String1:       constants.KUSD,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "GT",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionCollateralAmount,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionCollateralValue,
		String1:       constants.KUSD,
		String2:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionInterestRate,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionInterestRate,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.Error(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionInterestRate,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionCreditLineUsage,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))

	require.NoError(t, k.ValidateCondition(ctx, types.Condition{
		ConditionType: types.ConditionAutomationFundsAmount,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}))
}

func TestConditions2(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	c := types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "GTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, c))

	met, err := k.CheckIfConditionMet(ctx, accAddress, c, 0, 0)
	require.NoError(t, err)
	require.True(t, met)

	c = types.Condition{
		ConditionType: types.ConditionWalletAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDecWithPrec(1, 1),
		Comparison:    "LTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, c))

	met, err = k.CheckIfConditionMet(ctx, accAddress, c, 0, 0)
	require.NoError(t, err)
	require.False(t, met)
}

func TestConditions3(t *testing.T) {
	k, _, _, mmMsg, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	c := types.Condition{
		ConditionType: types.ConditionCollateralAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, c))

	met, err := k.CheckIfConditionMet(ctx, accAddress, c, 0, 0)
	require.NoError(t, err)
	require.False(t, met)

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &mmtypes.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "1000",
	}))

	met, err = k.CheckIfConditionMet(ctx, accAddress, c, 0, 0)
	require.NoError(t, err)
	require.True(t, met)
}

func TestConditions4(t *testing.T) {
	k, _, _, mmMsg, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	c1 := types.Condition{
		ConditionType: types.ConditionCollateralAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, c1))

	c2 := types.Condition{
		ConditionType: types.ConditionCollateralAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, c2))

	conditions := []types.Condition{c1, c2}
	_, numValid, err := k.CheckIfConditionsMet(ctx, accAddress, conditions, 0)
	require.NoError(t, err)
	require.NotEqual(t, len(conditions), numValid)

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &mmtypes.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "1000",
	}))

	_, numValid, err = k.CheckIfConditionsMet(ctx, accAddress, conditions, 0)
	require.NoError(t, err)
	require.Equal(t, len(conditions), numValid)
}

func TestConditions5(t *testing.T) {
	k, _, _, _, ctx := keepertest.SetupStrategiesMsgServer(t)

	require.Error(t, k.ValidateConditions(ctx, nil))
	require.Error(t, k.ValidateConditions(ctx, []types.Condition{}))

	require.NoError(t, k.ValidateConditions(ctx, []types.Condition{
		{
			ConditionType: types.ConditionPrice,
			String1:       constants.KUSD,
			String2:       constants.BaseCurrency,
			Value:         math.LegacyNewDecWithPrec(25, 2),
			Comparison:    "GT",
		},
	}))

	require.NoError(t, k.ValidateConditions(ctx, []types.Condition{
		{
			ConditionType: types.ConditionPrice,
			String1:       constants.KUSD,
			String2:       constants.BaseCurrency,
			Value:         math.LegacyNewDecWithPrec(25, 2),
			Comparison:    "GT",
		},
		{
			ConditionType: types.ConditionWalletAmount,
			String1:       constants.KUSD,
			Value:         math.LegacyNewDec(100),
			Comparison:    "GT",
		},
	}))
}

func TestConditions6(t *testing.T) {
	k, _, dexMsg, _, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	condition := types.Condition{
		ConditionType: types.ConditionLiquidityAmount,
		String1:       constants.BaseCurrency,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "GTE",
	}
	require.NoError(t, k.ValidateCondition(ctx, condition))

	met, err := k.CheckIfConditionMet(ctx, accAddress, condition, 0, 0)
	require.NoError(t, err)
	require.False(t, met)

	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.Equal(t, int64(1000), k.DexKeeper.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice).Int64())

	met, err = k.CheckIfConditionMet(ctx, accAddress, condition, 0, 0)
	require.NoError(t, err)
	require.True(t, met)
}

func TestConditions7(t *testing.T) {
	k, _, _, mmMsg, ctx := keepertest.SetupStrategiesMsgServer(t)
	accAddress, _ := sdk.AccAddressFromBech32(keepertest.Alice)

	require.NoError(t, keepertest.AddDeposit(ctx, mmMsg, &mmtypes.MsgAddDeposit{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000",
	}))

	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &mmtypes.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.BaseCurrency,
		Amount:  "10000",
	}))
	require.NoError(t, keepertest.AddCollateral(ctx, mmMsg, &mmtypes.MsgAddCollateral{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "10000",
	}))

	conditionIR := types.Condition{
		ConditionType: types.ConditionInterestRate,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDecWithPrec(6, 2),
		Comparison:    "LT",
	}
	require.NoError(t, k.ValidateCondition(ctx, conditionIR))

	conditionLA := types.Condition{
		ConditionType: types.ConditionLoanAmount,
		String1:       constants.KUSD,
		Value:         math.LegacyNewDec(1000),
		Comparison:    "LT",
	}
	require.NoError(t, k.ValidateCondition(ctx, conditionLA))

	met, err := k.CheckIfConditionMet(ctx, accAddress, conditionIR, 0, 0)
	require.NoError(t, err)
	require.True(t, met)

	met, err = k.CheckIfConditionMet(ctx, accAddress, conditionLA, 0, 0)
	require.NoError(t, err)
	require.True(t, met)

	require.NoError(t, keepertest.Borrow(ctx, mmMsg, &mmtypes.MsgBorrow{
		Creator: keepertest.Alice,
		Denom:   constants.KUSD,
		Amount:  "6000",
	}))

	met, err = k.CheckIfConditionMet(ctx, accAddress, conditionIR, 0, 0)
	require.NoError(t, err)
	require.False(t, met)

	met, err = k.CheckIfConditionMet(ctx, accAddress, conditionLA, 0, 0)
	require.NoError(t, err)
	require.False(t, met)
}
