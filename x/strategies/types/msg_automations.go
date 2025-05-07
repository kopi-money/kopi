package types

import (
	"encoding/json"
	"fmt"
	"regexp"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var RegexPercentage = regexp.MustCompile(`^(100|[1-9][0-9]?|0[1-9])%$`)

var (
	_ sdk.Msg = &MsgAutomationsAdd{}
	_ sdk.Msg = &MsgAutomationsImport{}
	_ sdk.Msg = &MsgAutomationsUpdate{}
	_ sdk.Msg = &MsgAutomationsRemove{}
	_ sdk.Msg = &MsgAutomationsActive{}
)

func (msg *MsgAutomationsAdd) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return validateAutomation(msg)
}

func (msg *MsgAutomationsImport) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	_, err := msg.Convert()
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	return nil
}

func (msg *MsgAutomationsImport) Convert() ([]MsgAutomationsAdd, error) {
	var automations []AutomationImport
	if err := json.Unmarshal([]byte(msg.Automations), &automations); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	var newAutomations []MsgAutomationsAdd
	for _, automation := range automations {
		conditions, err := json.Marshal(automation.Conditions)
		if err != nil {
			return nil, fmt.Errorf("marshal conditions: %w", err)
		}

		actions, err := json.Marshal(automation.Actions)
		if err != nil {
			return nil, fmt.Errorf("marshal conditions: %w", err)
		}

		newAutomations = append(newAutomations, MsgAutomationsAdd{
			Creator:        msg.Creator,
			Title:          automation.Title,
			IntervalType:   automation.IntervalType,
			IntervalLength: automation.IntervalLength,
			ValidityType:   automation.ValidityType,
			ValidityValue:  automation.ValidityValue,
			Conditions:     string(conditions),
			Actions:        string(actions),
		})
	}

	return newAutomations, nil
}

func (msg *MsgAutomationsUpdate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return validateAutomation(msg)
}

func (msg *MsgAutomationsRemove) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (msg *MsgAutomationsActive) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
