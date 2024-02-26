package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgUpdateFeeReimbursement{}
	_ sdk.Msg = &MsgUpdateMaxOrderLife{}
	_ sdk.Msg = &MsgUpdateReserveShare{}
	_ sdk.Msg = &MsgUpdateTradeFee{}
	_ sdk.Msg = &MsgUpdateVirtualLiquidityDecay{}
)

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUpdateFeeReimbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateMaxOrderLife) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateReserveShare) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateTradeFee) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateVirtualLiquidityDecay) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
