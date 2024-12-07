package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/strategies/types"
)

const (
	AutomationIntervalNotSet = iota
	AutomationIntervalSeconds
	AutomationIntervalMinutes
	AutomationIntervalHours
	AutomationIntervalDays
	AutomationIntervalWeeks
	AutomationIntervalMonths

	AutomationValidityUnlimited
	AutomationValidityNumExecutions
	AutomationValidityFeesConsumed
)

const (
	InactiveReasonAutomationFunds = iota
	InactiveReasonTimeValidity
	InactiveReasonFundsConsumed
	InactiveReasonTimesExecuted
	InactiveReasonError
)

func inactiveReason(reason int64) *types.InactiveReason {
	return &types.InactiveReason{Reason: reason}
}

func isValidValidity[I int | int64](validityType, validityValue I) error {
	switch validityType {
	case AutomationValidityUnlimited:
		if validityValue != 0 {
			return fmt.Errorf("value1 has to be 0")
		}

	case AutomationIntervalSeconds, AutomationIntervalMinutes,
		AutomationIntervalHours, AutomationIntervalDays,
		AutomationIntervalWeeks, AutomationIntervalMonths,
		AutomationValidityNumExecutions, AutomationValidityFeesConsumed:

		if validityValue <= 0 {
			return fmt.Errorf("value1 has to be larger than 0")
		}
	default:
		return fmt.Errorf("invalid validity type: %v", validityType)
	}

	return nil
}

func isTimeValidity(automation types.Automation) bool {
	switch automation.ValidityType {
	case AutomationIntervalSeconds, AutomationIntervalMinutes,
		AutomationIntervalHours, AutomationIntervalDays,
		AutomationIntervalWeeks, AutomationIntervalMonths:
		return true
	default:
		return false
	}
}

func checkTimeValidity(automation *types.Automation, blockHeight, blocksPerYear int64, blockTime time.Time) bool {
	validityLengthInSeconds, _ := convertIntervalLength(automation.ValidityType, automation.ValidityValue)

	if automation.PeriodStartTimestamp != nil {
		runtimeInSeconds := blockTime.Sub(*automation.PeriodStartTimestamp).Seconds()
		return int64(runtimeInSeconds) < validityLengthInSeconds
	} else {
		validityLengthInBlocks := convertSecondsToBlocks(validityLengthInSeconds, blocksPerYear)
		validUntil := automation.PeriodStart + validityLengthInBlocks
		return validUntil > blockHeight
	}
}

func checkValidity(automation types.Automation) (bool, *types.InactiveReason, error) {
	switch automation.ValidityType {
	case AutomationValidityUnlimited:
		return true, nil, nil
	case AutomationValidityNumExecutions:
		return automation.PeriodTimesExecuted < automation.ValidityValue, inactiveReason(InactiveReasonTimesExecuted), nil
	case AutomationValidityFeesConsumed:
		fees := automation.PeriodConditionFeesConsumed + automation.PeriodActionFeesConsumed
		return int64(fees) < automation.ValidityValue, inactiveReason(InactiveReasonFundsConsumed), nil
	case AutomationIntervalSeconds, AutomationIntervalMinutes,
		AutomationIntervalHours, AutomationIntervalDays,
		AutomationIntervalWeeks, AutomationIntervalMonths:
		return automation.Active, inactiveReason(InactiveReasonTimeValidity), nil

	default:
		return false, nil, fmt.Errorf("invalid validity type: %v", automation.ValidityType)
	}
}

func convertSecondsToBlocks(lengthInSeconds, blocksPerYear int64) int64 {
	secondsPerBlock := float64(constants.SecondsPerYear) / float64(blocksPerYear)

	return int64(float64(lengthInSeconds) * secondsPerBlock)
}

func convertIntervalLength[I int | int64](intervalType, value I) (int64, error) {
	switch intervalType {
	case AutomationIntervalSeconds:
		// nothing to do
		break
	case AutomationIntervalMinutes:
		value *= 60
	case AutomationIntervalHours:
		value *= 60 * 60
	case AutomationIntervalDays:
		value *= 60 * 60 * 24
	case AutomationIntervalWeeks:
		value *= 60 * 60 * 24 * 7
	case AutomationIntervalMonths:
		value *= 60 * 60 * 24 * 30
	case AutomationIntervalNotSet:
		fallthrough
	default:
		return 0, fmt.Errorf("invalid automation interval id: %v", intervalType)
	}

	return int64(value), nil
}

func convertIntervalLengthDec[I int | int64](intervalType, value I) (math.LegacyDec, error) {
	value64, err := convertIntervalLength(intervalType, value)
	if err != nil {
		return math.LegacyDec{}, nil
	}

	return math.LegacyNewDec(value64), nil
}
