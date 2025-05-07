package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgUpdateMaxOrderLife{}
	_ sdk.Msg = &MsgUpdateReserveShare{}
	_ sdk.Msg = &MsgUpdateTradeFee{}
	_ sdk.Msg = &MsgUpdatePriceIncreasingFactor{}
)

func (msg *MsgUpdateMaxOrderLife) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}

func (msg *MsgUpdateReserveShare) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := denomtypes.IsDec(msg.ReserveShare, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("reserve_share: %w", err)
	}

	return nil
}

func (msg *MsgUpdateTradeFee) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := denomtypes.IsDec(msg.TradeFee, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("trade_fee: %w", err)
	}

	return nil
}

func (msg *MsgUpdatePriceIncreasingFactor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := denomtypes.IsDec(msg.PriceIncreasingFactor, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("price_increasing_factor: %w", err)
	}

	return nil
}
