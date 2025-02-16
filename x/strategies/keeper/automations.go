package keeper

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kopi-money/kopi/constants"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

// SetAutomation sets a specific liquidity in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetAutomation(ctx context.Context, automation types.Automation) types.Automation {
	if automation.Index == 0 {
		nextIndex, _ := k.automationsNextIndex.Get(ctx)
		nextIndex++
		automation.Index = nextIndex

		k.SetAutomationsNextIndex(ctx, nextIndex)
	}

	k.automations.Set(ctx, automation.Index, automation)
	return automation
}

func (k Keeper) SetAutomations(ctx context.Context, automations []*types.Automation) error {
	for _, automation := range automations {
		if automation == nil {
			return fmt.Errorf("automation was nil")
		}

		k.automations.Set(ctx, automation.Index, *automation)
	}

	return nil
}

func (k Keeper) GetAutomationsNextIndex(ctx context.Context) (uint64, bool) {
	return k.automationsNextIndex.Get(ctx)
}

func (k Keeper) SetAutomationsNextIndex(ctx context.Context, nextIndex uint64) {
	k.automationsNextIndex.Set(ctx, nextIndex)
}

func (k Keeper) AutomationIterator(ctx context.Context) cache.Iterator[uint64, types.Automation] {
	return k.automations.Iterator(ctx, nil)
}

func (k Keeper) AutomationCacheIterator(ctx context.Context) cache.Iterator[uint64, types.Automation] {
	return k.automations.CacheIterator(ctx)
}

func (k Keeper) GetAutomations(ctx context.Context) (list []*types.Automation) {
	iterator := k.AutomationIterator(ctx)
	for iterator.Valid() {
		automation := iterator.GetNext()
		list = append(list, &automation)
	}

	return
}

func (k Keeper) GetAutomationsByAddress(ctx context.Context, address string) (list []*types.Automation) {
	iterator := k.AutomationIterator(ctx)
	for iterator.Valid() {
		automation := iterator.GetNext()
		if automation.Address == address {
			list = append(list, &automation)
		}
	}

	return
}

func (k Keeper) HandleAutomations(ctx context.Context) error {
	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	blocksPerYearDec, err := k.BlockspeedKeeper.BlocksPerYear(ctx)
	if err != nil {
		return fmt.Errorf("could not get blocks per year: %w", err)
	}

	blocksPerYear := blocksPerYearDec.RoundInt64()
	secondsPerBlock := k.BlockspeedKeeper.GetSecondsPerBlock(ctx)
	params := k.GetParams(ctx)

	var (
		totalConsumption uint64 = 0
		isBelowCheckRate bool
	)

	iterator := k.AutomationCacheIterator(ctx)
	for automationExecutionIndex, entry := range iterator.GetAllFromCache() {
		if entry.Value().Value() == nil {
			k.Logger().Error("nil value")
			continue
		}

		automation := *entry.Value().Value()
		if !automation.Active {
			continue
		}

		if !k.handleTimeValidity(ctx, automation, blockHeight, blocksPerYear) {
			continue
		}

		isBelowCheckRate, err = k.checkAutomationBelowCheckRate(ctx, secondsPerBlock, automation, blockHeight)
		if err != nil {
			k.Logger().Error(err.Error())
			continue
		}

		if !isBelowCheckRate {
			continue
		}

		if _, _, err = k.HandleAutomation(ctx, params, automation, automationExecutionIndex, &totalConsumption); err != nil {
			k.Logger().Error(err.Error())
			continue
		}
	}

	if totalConsumption > 0 {
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, math.NewInt(int64(totalConsumption))))
			if err = k.BankKeeper.SendCoinsFromModuleToModule(innerCtx, types.PoolAutomationFunds, dextypes.PoolReserve, coins); err != nil {
				return fmt.Errorf("could not send funds from funds pool to reserve: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// checkAutomationBelowCheckRate calculates the rate with which the automation has been executed in relation to how many times
// it should have been executed given the time it has been active.
func (k Keeper) checkAutomationBelowCheckRate(ctx context.Context, secondsPerBlock math.LegacyDec, automation types.Automation, blockHeight int64) (bool, error) {
	_, _, expectedChecks, err := k.getIntervalCheckData(ctx, secondsPerBlock, automation, blockHeight)
	if err != nil {
		return false, err
	}

	return math.LegacyNewDec(automation.PeriodTimesChecked).LT(expectedChecks), nil
}

func (k Keeper) getIntervalCheckData(ctx context.Context, secondsPerBlock math.LegacyDec, automation types.Automation, blockHeight int64) (math.LegacyDec, math.LegacyDec, math.LegacyDec, error) {
	intervalInSeconds, err := convertIntervalLengthDec(automation.IntervalType, automation.IntervalLength)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("could not convert interval length: %w", err)
	}

	var runtimeInSeconds math.LegacyDec
	if automation.PeriodStartTimestamp != nil {
		now := sdk.UnwrapSDKContext(ctx).BlockTime()
		runtimeInSeconds = math.LegacyNewDec(int64(now.Sub(*automation.PeriodStartTimestamp).Seconds()))
	} else {
		runtimeInBlocks := blockHeight - automation.PeriodStart
		runtimeInSeconds = convertBlocksToSeconds(secondsPerBlock, runtimeInBlocks)
	}

	expectedChecks := runtimeInSeconds.Quo(intervalInSeconds) // C

	return intervalInSeconds, runtimeInSeconds, expectedChecks, nil
}

func convertBlocksToSeconds(secondsPerBlock math.LegacyDec, numBlocks int64) math.LegacyDec {
	numBlocksDec := math.LegacyNewDec(numBlocks)
	return numBlocksDec.Mul(secondsPerBlock)
}

func (k Keeper) handleTimeValidity(ctx context.Context, automation types.Automation, blockHeight, blocksPerYear int64) bool {
	if !isTimeValidity(automation) {
		return true
	}

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	automation.Active = checkTimeValidity(&automation, blockHeight, blocksPerYear, blockTime)

	if !automation.Active {
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
			sdk.NewEvent(
				"automation_validity",
				sdk.Attribute{Key: "automation_index", Value: strconv.Itoa(int(automation.Index))},
				sdk.Attribute{Key: "validity_type", Value: strconv.Itoa(int(automation.ValidityType))},
				sdk.Attribute{Key: "address", Value: automation.Address},
			),
		)

		_ = cache.Transact(ctx, func(innerCtx context.Context) error {
			k.automations.Set(innerCtx, automation.Index, automation)
			return nil
		})
	}

	return automation.Active
}

func (k Keeper) HandleAutomation(ctx context.Context, params types.Params, automation types.Automation, automationExecutionIndex int, totalConsumption *uint64) (bool, []bool, error) {
	automation.PeriodTimesChecked++

	conditionsMatched, successfulActions, err := k.handleAutomation(ctx, params, &automation, automationExecutionIndex, totalConsumption)
	if err != nil {
		k.Logger().Error(fmt.Sprintf("k.handleAutomation: %v", err.Error()))
	}

	// The automation might have been set inactive because there are not enough funds for executing it
	if automation.Active {
		automation.Active, automation.InactiveReason, err = checkValidity(automation)
	}

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.automations.Set(innerCtx, automation.Index, automation)
		return nil
	})

	return conditionsMatched, successfulActions, nil
}

func (k Keeper) handleAutomation(ctx context.Context, params types.Params, automation *types.Automation, automationExecutionIndex int, totalConsumption *uint64) (bool, []bool, error) {
	acc, _ := sdk.AccAddressFromBech32(automation.Address)

	cost := k.determineAutomationCost(ctx, automation)
	funds := k.GetAutomationFunds(ctx, acc.String())

	defer func() {
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
			sdk.NewEvent("automation_executed",
				sdk.Attribute{Key: "automation_index", Value: strconv.Itoa(int(automation.Index))},
				sdk.Attribute{Key: "funds", Value: strconv.Itoa(int(funds.Int64()))},
				sdk.Attribute{Key: "cost", Value: strconv.Itoa(int(cost.Int64()))},
			),
		)
	}()

	if funds.LT(cost) {
		automation.Active = false
		automation.InactiveReason = inactiveReason(InactiveReasonAutomationFunds)
		return false, nil, nil
	}

	numCheckedConditions, numValidConditions, err := k.CheckIfConditionsMet(ctx, acc, automation.Conditions, int(automation.Index))
	if err != nil {
		return false, nil, fmt.Errorf("could not check whether conditions are met: %w", err)
	}

	if numCheckedConditions == 0 {
		return false, nil, nil
	}

	conditionCheckCost := uint64(numCheckedConditions) * k.GetAutomationFeeCondition(ctx)
	if err = k.consumeAutomationFunds(ctx, acc, conditionCheckCost, totalConsumption); err != nil {
		return false, nil, err
	}

	automation.PeriodConditionFeesConsumed += conditionCheckCost
	automation.TotalConditionFeesConsumed += conditionCheckCost

	var (
		allConditionsMatched bool
		successfulActions    []bool
	)

	if numValidConditions == len(automation.Conditions) {
		automation.PeriodTimesExecuted++
		automation.TotalTimesExecuted++
		allConditionsMatched = true

		var (
			executed bool
			success  bool
		)
		for actionIndex, action := range automation.Actions {
			if err = cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
				err = k.ExecuteAction(innerCtx, acc, action, int(automation.Index), automationExecutionIndex, actionIndex)
				success = err == nil

				// If the occurred error belongs to the list of whitelisted errors, the execution fee still is applied
				if err == nil || errorIsOf(err, types.ValidErrors) {
					executed = true
				}

				return err
			}); err != nil {
				if !errorIsOf(err, types.ValidErrors) {
					automation.InactiveReason = inactiveReason(InactiveReasonError)
					automation.Active = false
				}
			}

			// successful actions are tracked for unit test purposes
			successfulActions = append(successfulActions, success)

			if executed {
				actionCost := getActionCost(params, action.ActionType)
				automation.PeriodActionFeesConsumed += actionCost
				automation.TotalActionFeesConsumed += actionCost

				if err = k.consumeAutomationFunds(ctx, acc, actionCost, totalConsumption); err != nil {
					return allConditionsMatched, successfulActions, err
				}
			}
		}
	}

	return allConditionsMatched, successfulActions, nil
}

func (k Keeper) determineAutomationCost(ctx context.Context, automation *types.Automation) math.Int {
	cost := int64(k.GetParams(ctx).AutomationFeeCondition) * int64(len(automation.Conditions))
	cost += k.getActionsCost(ctx, automation.Actions)
	return math.NewInt(cost)
}
