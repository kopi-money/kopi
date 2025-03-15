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
	_ sdk.Msg = &MsgAddLiquidity{}
	_ sdk.Msg = &MsgRemoveAllLiquidityForDenom{}
	_ sdk.Msg = &MsgRemoveLiquidity{}
)

func (msg *MsgAddLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.ValidateDenomName(msg.Denom); err != nil {
		return err
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	return nil
}

func (msg *MsgRemoveAllLiquidityForDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.ValidateDenomName(msg.WithdrawDenom); err != nil {
		return err
	}

	if err := denomtypes.ValidateDenomName(msg.PayoutDenom); err != nil {
		return err
	}

	return nil
}

func (msg *MsgRemoveLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if err := denomtypes.ValidateDenomName(msg.WithdrawDenom); err != nil {
		return err
	}

	if err := denomtypes.ValidateDenomName(msg.PayoutDenom); err != nil {
		return err
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	return nil
}
