package types

import (
	"encoding/json"
	"fmt"
	factorytypes "github.com/kopi-money/kopi/x/tokenfactory/types"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

type AutomationMessage interface {
	GetIntervalType() string
	GetIntervalLength() string
	GetValidityType() string
	GetValidityValue() string
	GetTitle() string
	GetConditions() string
	GetActions() string
}

func validateAutomation(am AutomationMessage) error {
	if _, err := strconv.Atoi(am.GetIntervalType()); err != nil {
		return fmt.Errorf("invalid interval type: %v", am.GetIntervalType())
	}

	if _, err := strconv.Atoi(am.GetIntervalLength()); err != nil {
		return fmt.Errorf("invalid interval length: %v", am.GetIntervalLength())
	}

	if _, err := strconv.Atoi(am.GetValidityType()); err != nil {
		return fmt.Errorf("invalid validity type: %v", am.GetValidityType())
	}

	if err := denomtypes.IsDec(am.GetValidityValue(), math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("validity_value: %w", err)
	}

	if len(am.GetTitle()) == 0 {
		return ErrAutomationTitleEmpty
	}

	if len(am.GetTitle()) > 30 {
		return ErrAutomationTitleTooLong
	}

	var messageConditions []MessageCondition
	if err := json.Unmarshal([]byte(am.GetConditions()), &messageConditions); err != nil {
		return fmt.Errorf("unmarshal conditions: %w", err)
	}

	if len(messageConditions) > 16 {
		return fmt.Errorf("must not contain more than 16 conditions")
	}

	if _, err := ConvertConditions(messageConditions); err != nil {
		return fmt.Errorf("convert conditions: %w", err)
	}

	var actions []*Action
	if err := json.Unmarshal([]byte(am.GetActions()), &actions); err != nil {
		return fmt.Errorf("unmarshal actions: %w", err)
	}

	if len(messageConditions) > 16 {
		return fmt.Errorf("must not contain more than 16 actions")
	}

	if err := checkActions(actions); err != nil {
		return err
	}

	return nil
}

func checkAutomationString(value string) error {
	if value == "" {
		return nil
	}

	if _, err := sdk.AccAddressFromBech32(value); err == nil {
		return nil
	}

	if _, err := sdk.ValAddressFromBech32(value); err == nil {
		return nil
	}

	if err := denomtypes.ValidateDenomName(value); err == nil {
		return nil
	}

	if err := factorytypes.ValidateDenomName(value); err == nil {
		return nil
	}

	return fmt.Errorf("given string was neiter denom nor address")
}

func checkAmountString(amountString string) error {
	if amountString == "" {
		return nil
	}

	if RegexPercentage.Match([]byte(amountString)) {
		return nil
	}

	value, ok := math.NewIntFromString(amountString)
	if !ok {
		return fmt.Errorf("invalid amount, was: %v", amountString)
	}

	if value.IsZero() {
		return fmt.Errorf("value must not be zero")
	}

	return nil
}
