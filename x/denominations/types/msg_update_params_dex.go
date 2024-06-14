package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgAddDEXDenom{}
	_ sdk.Msg = &MsgUpdateDEXDenomMinimumLiquidity{}
	_ sdk.Msg = &MsgUpdateDEXDenomMinimumOrderSize{}
)

func (m *MsgAddDEXDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateDEXDenomMinimumLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateDEXDenomMinimumOrderSize) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
