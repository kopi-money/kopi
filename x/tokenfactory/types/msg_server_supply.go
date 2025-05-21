package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgMintDenom{}
	_ sdk.Msg = &MsgBurnDenom{}
)

func (msg *MsgMintDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.TargetAddress); err != nil {
		return errorsmod.Wrap(err, "invalid target address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	return nil
}

func (msg *MsgBurnDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	return nil
}
