package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgAddCollateralDenom{}
	_ sdk.Msg = &MsgUpdateCollateralDenomMaxDeposit{}
	_ sdk.Msg = &MsgUpdateCollateralDenomLTV{}
)

func (m *MsgAddCollateralDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateCollateralDenomMaxDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateCollateralDenomLTV) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
