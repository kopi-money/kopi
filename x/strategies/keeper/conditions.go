package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) CheckConditions(ctx context.Context, conditions []types.Condition) error {
	if len(conditions) == 0 {
		return types.ErrEmptyConditions
	}

	for conditionIndex, condition := range conditions {
		if err := k.CheckCondition(ctx, condition); err != nil {
			return fmt.Errorf("could not convert condition[%d]: %w", conditionIndex, err)
		}
	}

	return nil
}

func (k Keeper) CheckCondition(ctx context.Context, condition types.Condition) error {
	if !types.IsValidComparison(condition.ConditionType, condition.Comparison) {
		return fmt.Errorf("invalid comparison: %v", condition.Comparison)
	}

	if condition.Value.IsNil() {
		return fmt.Errorf("condition value must not be nil")
	}

	// PriceChangePercentage is the only condition that can have a negative value since a change in % can be negative
	if !(condition.ConditionType == types.ConditionPriceChangePercentage || condition.ConditionType == types.ConditionPriceChangeAmount) {
		if condition.Value.IsNegative() {
			return fmt.Errorf("must not be less than 0, was: %v", condition.Value.String())
		}
	}

	switch condition.ConditionType {
	case types.ConditionAutomationFundsAmount, types.ConditionStakingRewardsAmount:
		if condition.String1 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if condition.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

	case types.ConditionPrice, types.ConditionWalletValue, types.ConditionLiquidityValue:
		if !k.DenomKeeper.IsValidDenom(ctx, condition.String1) {
			return fmt.Errorf("invalid string1: %v", condition.String1)
		}

		if !k.DenomKeeper.IsValidDenom(ctx, condition.String2) {
			return fmt.Errorf("invalid string2: %v", condition.String2)
		}

		if condition.String1 == condition.String2 {
			return fmt.Errorf("same denom twice")
		}

	case types.ConditionCollateralAmount:
		if condition.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if !k.DenomKeeper.IsCollateralDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no collateral denom")
		}

	case types.ConditionCollateralValue:
		if !k.DenomKeeper.IsCollateralDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no collateral denom")
		}

		if !k.DenomKeeper.IsValidDenom(ctx, condition.String2) {
			return fmt.Errorf("string2 is no valid denom")
		}

	case types.ConditionLoanAmount, types.ConditionBorrowableAmount:
		if condition.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if !k.DenomKeeper.IsBorrowableDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no borrowable denom")
		}

	case types.ConditionLoanValue:
		if !k.DenomKeeper.IsBorrowableDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no borrowable denom")
		}

		if !k.DenomKeeper.IsValidDenom(ctx, condition.String2) {
			return fmt.Errorf("string2 is no valid denom")
		}

	case types.ConditionInterestRate:
		if condition.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if _, err := k.DenomKeeper.GetCAsset(ctx, condition.String1); err != nil {
			return fmt.Errorf("string1 is no borrowable denom")
		}

	case types.ConditionWalletAmount, types.ConditionLiquidityAmount:
		if !k.DenomKeeper.IsValidDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no valid denom")
		}

	case types.ConditionCreditLineUsage:
		if condition.String1 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if condition.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if condition.Value.IsNegative() {
			return fmt.Errorf("credit line usage must not be lower than 0")
		}

		if condition.Value.GT(math.LegacyOneDec()) {
			return fmt.Errorf("credit line usage must not be larger than 1")
		}

	case types.ConditionPriceChangePercentage:
		if condition.Value.LT(math.LegacyOneDec().Neg()) {
			return fmt.Errorf("percentage change cannot be less than -1 (ie less than -100%%)")
		}

		fallthrough
	case types.ConditionPriceChangeAmount:
		if !k.DenomKeeper.IsValidDenom(ctx, condition.String1) {
			return fmt.Errorf("string1 is no valid denom")
		}

		if condition.ReferencePrice == nil || condition.ReferencePrice.IsNil() {
			return fmt.Errorf("reference price must not be null")
		}

		if condition.ReferencePrice.IsNegative() {
			return fmt.Errorf("reference price must not be smaller than 0")
		}

	default:
		return fmt.Errorf("invalid condition type: %v", condition.ConditionType)
	}

	return nil
}

func (k Keeper) CheckIfConditionsMet(ctx context.Context, acc sdk.AccAddress, conditions []types.Condition, automationIndex int) (int, int, error) {
	checked := 0
	correct := 0

	for index, condition := range conditions {
		checked++

		conditionMet, err := k.CheckIfConditionMet(ctx, acc, condition, automationIndex, index)
		if err != nil {
			return 0, 0, fmt.Errorf("error checking whether condition index %v is met: %w", index, err)
		}

		if !conditionMet {
			return checked, correct, nil
		}

		correct++
	}

	return checked, correct, nil
}

func (k Keeper) CheckIfConditionMet(ctx context.Context, accAddr sdk.AccAddress, condition types.Condition, automationIndex, conditionIndex int) (bool, error) {
	var (
		conditionValue      = condition.Value
		conditionComparison = condition.Comparison
		value               math.LegacyDec
		err                 error
	)

	switch condition.ConditionType {
	case types.ConditionPrice:
		value, err = k.DexKeeper.CalculatePrice(ctx, condition.String1, condition.String2)
		if err != nil {
			return false, fmt.Errorf("could not calculate price: %w", err)
		}

	case types.ConditionPriceChangeAmount:
		if condition.Comparison == types.ComparisonIncreasedBy {
			conditionValue = condition.ReferencePrice.Add(condition.Value)
			conditionComparison = types.ComparisonGreaterThan
		} else {
			conditionValue = condition.ReferencePrice.Sub(condition.Value)
			conditionComparison = types.ComparisonLessThan
		}

		value, err = k.DexKeeper.GetPriceInUSD(ctx, condition.String1)
		if err != nil {
			return false, fmt.Errorf("could not calculate price: %w", err)
		}

		if value.IsZero() {
			return false, fmt.Errorf("value is zero")
		}

		value = math.LegacyOneDec().Quo(value) // C

	case types.ConditionPriceChangePercentage:
		var factor math.LegacyDec
		if condition.Comparison == types.ComparisonIncreasedBy {
			factor = math.LegacyOneDec().Add(condition.Value)
			conditionComparison = types.ComparisonGreaterThan
		} else {
			factor = math.LegacyOneDec().Sub(condition.Value)
			conditionComparison = types.ComparisonLessThan
		}

		conditionValue = condition.ReferencePrice.Mul(factor)

		value, err = k.DexKeeper.GetPriceInUSD(ctx, condition.String1)
		if err != nil {
			return false, fmt.Errorf("could not calculate price: %w", err)
		}

		if value.IsZero() {
			return false, fmt.Errorf("value is zero")
		}

		value = math.LegacyOneDec().Quo(value) // C

	case types.ConditionWalletAmount:
		value = k.BankKeeper.SpendableCoin(ctx, accAddr, condition.String1).Amount.ToLegacyDec()

	case types.ConditionWalletValue:
		coins := k.BankKeeper.SpendableCoin(ctx, accAddr, condition.String1)
		value, err = k.DexKeeper.GetValueIn(ctx, condition.String1, condition.String2, coins.Amount.ToLegacyDec())
		if err != nil {
			return false, fmt.Errorf("could not calculate value of wallet amount: %w", err)
		}

	case types.ConditionCollateralAmount:
		value = k.MMKeeper.GetCollateralForDenomForAddressWithDefault(ctx, condition.String1, accAddr.String()).ToLegacyDec()

	case types.ConditionCollateralValue:
		amount := k.MMKeeper.GetCollateralForDenomForAddressWithDefault(ctx, condition.String1, accAddr.String()).ToLegacyDec()
		value, err = k.DexKeeper.GetValueIn(ctx, condition.String1, condition.String2, amount)
		if err != nil {
			return false, fmt.Errorf("could not calculate value of collateral amount: %w", err)
		}

	case types.ConditionLiquidityAmount:
		value = k.DexKeeper.GetLiquidityByAddress(ctx, condition.String1, accAddr.String()).ToLegacyDec()

	case types.ConditionLiquidityValue:
		amount := k.DexKeeper.GetLiquidityByAddress(ctx, condition.String1, accAddr.String()).ToLegacyDec()
		value, err = k.DexKeeper.GetValueIn(ctx, condition.String1, condition.String2, amount)
		if err != nil {
			return false, fmt.Errorf("could not calculate value of liquidity amount: %w", err)
		}

	case types.ConditionInterestRate:
		value = k.MMKeeper.CalculateInterestRateForDenom(ctx, condition.String1)

	case types.ConditionLoanAmount:
		value = k.MMKeeper.GetLoanValue(ctx, condition.String1, accAddr.String())

	case types.ConditionLoanValue:
		amount := k.MMKeeper.GetLoanValue(ctx, condition.String1, accAddr.String())
		value, err = k.DexKeeper.GetValueIn(ctx, condition.String1, condition.String2, amount)
		if err != nil {
			return false, fmt.Errorf("could not calculate value of loan: %w", err)
		}

	case types.ConditionCreditLineUsage:
		value, err = k.MMKeeper.CalculateCreditLineUsage(ctx, accAddr.String())
		if err != nil {
			return false, fmt.Errorf("could not calculate credit line usage: %w", err)
		}

	case types.ConditionAutomationFundsAmount:
		value = k.GetAutomationFunds(ctx, accAddr.String()).ToLegacyDec()

	case types.ConditionBorrowableAmount:
		value, err = k.MMKeeper.CalculateBorrowableAmount(ctx, accAddr.String(), condition.String1)

	case types.ConditionStakingRewardsAmount:
		value, err = k.getStakingRewards(ctx, accAddr)

	default:
		return false, fmt.Errorf("invalid condition type: %v", condition.ConditionType)
	}

	matched, err := compare(conditionComparison, value, conditionValue)
	if err != nil {
		return false, fmt.Errorf("could not execute condition comparison: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"automation_condition_check",
			sdk.Attribute{Key: "automation_index", Value: strconv.Itoa(automationIndex)},
			sdk.Attribute{Key: "condition_index", Value: strconv.Itoa(conditionIndex)},
			sdk.Attribute{Key: "value", Value: value.String()},
		),
	)

	return matched, nil
}

func compare(mode string, v1, v2 math.LegacyDec) (bool, error) {
	switch mode {
	case types.ComparisonLessThan:
		return v1.LT(v2), nil
	case types.ComparisonLessThanEquals:
		return v1.LTE(v2), nil
	case types.ComparisonGreaterThan:
		return v1.GT(v2), nil
	case types.ComparisonGreaterThanEquals:
		return v1.GTE(v2), nil
	default:
		return false, fmt.Errorf("invalid compare mode: %v", mode)
	}
}
