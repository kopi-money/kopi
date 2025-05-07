package types

import (
	"cosmossdk.io/math"
	"fmt"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

type TradeMessage interface {
	GetAmount() string
	GetDenomGiving() string
	GetDenomReceiving() string
	getMaxPrice() *MaxPrice
}

func validateTradeData(msg TradeMessage) error {
	if err := denomtypes.IsInt(msg.GetAmount(), math.ZeroInt()); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if msg.getMaxPrice() != nil {
		if err := denomtypes.IsDec(msg.getMaxPrice().MaxPrice, math.LegacyZeroDec()); err != nil {
			return fmt.Errorf("max_price: %w", err)
		}
	}

	if err := denomtypes.ValidateDenomName(msg.GetDenomGiving()); err != nil {
		return fmt.Errorf("invalid denom_giving: %w", err)
	}

	if err := denomtypes.ValidateDenomName(msg.GetDenomReceiving()); err != nil {
		return fmt.Errorf("invalid denom_receiving: %w", err)
	}

	return nil
}
