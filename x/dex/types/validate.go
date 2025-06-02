package types

import (
	"fmt"

	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

type TradeMessage interface {
	GetAmount() string
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

	return nil
}
