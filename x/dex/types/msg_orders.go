package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgAddOrder{}
	_ sdk.Msg = &MsgRemoveOrder{}
	_ sdk.Msg = &MsgRemoveOrders{}
	_ sdk.Msg = &MsgUpdateOrder{}
)

func (msg *MsgAddOrder) getMaxPrice() *MaxPrice {
	return &MaxPrice{
		MaxPrice:    msg.MaxPrice,
		FeeIncluded: false,
	}
}

func (msg *MsgAddOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.TradeAmount != "" {
		if err := denomtypes.IsInt(msg.TradeAmount, math.ZeroInt()); err != nil {
			return fmt.Errorf("trade_amount: %w", err)
		}
	}

	return validateTradeData(msg)
}

func (msg *MsgRemoveOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}

func (msg *MsgRemoveOrders) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}

func (msg *MsgUpdateOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := denomtypes.IsInt(msg.TradeAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("trade_amount: %w", err)
	}

	if err := denomtypes.IsDec(msg.MaxPrice, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("max_price: %w", err)
	}

	return nil
}
