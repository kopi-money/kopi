package types

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/measurement"
	"github.com/kopi-money/kopi/trading"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TradeType int

const (
	TradeTypeSell = iota + 1
	TradeTypeBuy
)

type TradeContext struct {
	context.Context

	TradeType TradeType

	TradeAmount            math.Int
	MaxPrice               *trading.MaxPriceData
	MinimumTradeAmount     *math.Int
	MaximumAvailableAmount math.Int
	Fee                    *math.LegacyDec

	CalcMaximumTradableAmountByLiquidity trading.CalculateMaximumAmountByLiquidity
	CalcMaximumTradableAmountByPrice     trading.CalculateMaximumAmountByPrice
	CalcMaximumTradableAmountByWallet    func(trading.Liquidity, trading.Liquidity, math.LegacyDec) math.Int

	TradeDenomGiving    string
	TradeDenomReceiving string

	ExcludeFromDiscount bool
	ProtocolTrade       bool
	IsOrder             bool

	CoinSource      string
	CoinTarget      string
	DiscountAddress string

	TradeBalances TradeBalances
	OrdersCaches  *OrdersCaches

	FlatPrice   trading.ConstantProductTrade
	Measurement *measurement.Measurement
}

func (tc *TradeContext) FeeDenom() string {
	if tc.IsBuy() {
		return tc.TradeDenomGiving
	} else {
		return tc.TradeDenomReceiving
	}
}

func (tc *TradeContext) IsBuy() bool {
	return tc.TradeType == TradeTypeBuy
}

func (tc *TradeContext) CoinTargetForEffective() *string {
	if tc.ProtocolTrade {
		return nil
	}

	return &tc.CoinTarget
}

func (tc *TradeContext) GetOrdersCaches() *OrdersCaches {
	return tc.OrdersCaches
}

type TradeSimulationResult struct {
	AmountIntermediate math.Int
	AmountGiven        math.Int
	AmountReceived     math.Int
	FeeGiven           math.Int
}

func (tsr TradeSimulationResult) PricePaid() (math.LegacyDec, error) {
	if !tsr.AmountReceived.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("amount received not positive")
	}

	return tsr.AmountGiven.ToLegacyDec().Quo(tsr.AmountReceived.ToLegacyDec()), nil
}

type Sender interface {
	SendCoins(ctx context.Context, address sdk.AccAddress, accAddress sdk.AccAddress, coins sdk.Coins) error
}

type TradeBalances interface {
	AddTransfer(string, string, string, math.Int)
	NetBalance(string, string) math.Int
	Settle(context.Context, Sender) error
}

type TradeCalculation interface {
	// Forward is for sells, ie users give a fixed amount and receive a calculated amount
	Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int
	// Backward is for buys, ie users give receive fixed amount and give a calculated amount
	Backward(poolFrom, poolTo, result math.LegacyDec) math.Int
}
