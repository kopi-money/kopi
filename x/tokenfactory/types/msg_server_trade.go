package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgSell{}
	_ sdk.Msg = &MsgBuy{}
	_ sdk.Msg = &MsgBuyback{}
)

func (msg *MsgSell) ValidateBasic() error {
	return validateTradeMessage(msg)
}

func (msg *MsgBuy) ValidateBasic() error {
	return validateTradeMessage(msg)
}

func (msg *MsgBuyback) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.BuybackAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("buyback_amount: %w", err)
	}

	return nil
}

func validateTradeMessage(msg MsgTrade) error {
	if _, err := sdk.AccAddressFromBech32(msg.GetCreator()); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.GetFullFactoryDenomName()); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.GetAmount(), math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if maxPrice := msg.GetMaxPrice(); maxPrice != nil {
		if err := denomtypes.IsDec(maxPrice.MaxPrice, math.LegacyZeroDec()); err != nil {
			return fmt.Errorf("max_price: %w", err)
		}
	}

	return nil
}

type MsgTrade interface {
	GetAmount() string
	GetCreator() string
	GetDenomGiving() string
	GetDenomReceiving() string
	GetFullFactoryDenomName() string
	GetMaxPrice() *MaxPrice
}
