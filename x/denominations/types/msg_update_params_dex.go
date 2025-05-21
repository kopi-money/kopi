package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgDexAddDenom{}
	_ sdk.Msg = &MsgDexUpdateMinimumTradeLiquidity{}
	_ sdk.Msg = &MsgDexUpdateMinimumDexLiquidity{}
	_ sdk.Msg = &MsgDexUpdateMinimumOrderSize{}
)

func (msg *MsgDexAddDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (msg *MsgDexUpdateMinimumTradeLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := IsInt(msg.MinimumTradeLiquidity, math.ZeroInt()); err != nil {
		return fmt.Errorf("min_liquidity: %w", err)
	}

	if err := ValidateDenomName(msg.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	return nil
}

func (msg *MsgDexUpdateMinimumDexLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := IsInt(msg.MinimumDexLiquidity, math.ZeroInt()); err != nil {
		return fmt.Errorf("min_liquidity: %w", err)
	}

	if err := ValidateDenomName(msg.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	return nil
}

func (msg *MsgDexUpdateMinimumOrderSize) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := IsInt(msg.MinOrderSize, math.ZeroInt()); err != nil {
		return fmt.Errorf("min_order_size: %w", err)
	}

	if err := ValidateDenomName(msg.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	return nil
}
