package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgAddKCoin{}
	_ sdk.Msg = &MsgUpdateKCoinSupply{}
	_ sdk.Msg = &MsgUpdateKCoinMintAmount{}
	_ sdk.Msg = &MsgUpdateKCoinBurnAmount{}
	_ sdk.Msg = &MsgAddKCoinReferences{}
	_ sdk.Msg = &MsgRemoveKCoinReferences{}
)

func (m *MsgAddKCoin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateKCoinSupply) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateKCoinMintAmount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgUpdateKCoinBurnAmount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgAddKCoinReferences) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (m *MsgRemoveKCoinReferences) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
