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
	_ sdk.Msg = &MsgBuy{}
	_ sdk.Msg = &MsgSell{}
)

func (msg *MsgBuy) getMaxPrice() *MaxPrice {
	return msg.MaxPrice
}

func (msg *MsgBuy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.IsInt(msg.MinimumTradeAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("minimum_trade_amount: %w", err)
	}

	return validateTradeData(msg)
}

func (msg *MsgSell) getMaxPrice() *MaxPrice {
	return msg.MaxPrice
}

func (msg *MsgSell) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.IsInt(msg.MinimumTradeAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("minimum_trade_amount: %w", err)
	}

	return validateTradeData(msg)
}
