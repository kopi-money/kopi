package types

import (
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/trading"
)

func (tl TradeLiquidity) GetFull() math.LegacyDec {
	return tl.Actual.Add(tl.Virtual)
}

// AdjustToTradeValue determines what of the actual liquidity can be used and how much has to be filled using virtual.
func (tl TradeLiquidity) AdjustToTradeValue(globalTradeValueBase math.LegacyDec) trading.Liquidity {
	share := globalTradeValueBase.Quo(tl.TradeValueBase)
	tradeValue := tl.TradeValue.Mul(share)

	actualToUse := math.LegacyMinDec(tl.Actual, tradeValue)
	virtualToUSe := tradeValue.Sub(actualToUse)

	return trading.Liquidity{
		Actual:  actualToUse,
		Virtual: virtualToUSe,
	}
}
