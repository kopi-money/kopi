package types

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/trading"
)

type TradeContext struct {
	context.Context

	Callbacks          trading.Callbacks
	TradeAmount        math.Int
	MaxPrice           *trading.MaxPriceData
	MinimumTradeAmount *math.Int

	Pool           LiquidityPool
	DenomGiving    string
	DenomReceiving string
	Creator        string
	FeeIncluded    bool

	CPTrade trading.ConstantProductTrade
}

func (tc TradeContext) ToTradeData() trading.TradeData {
	getFeeAmount := trading.GetSellFee
	feeStepOne := false

	if tc.Callbacks.IsSell() && tc.DenomGiving == tc.Pool.KCoin {
		getFeeAmount = trading.GetSellFee
		feeStepOne = true
	}

	if !tc.Callbacks.IsSell() && tc.DenomGiving != tc.Pool.KCoin {
		getFeeAmount = trading.GetBuyFee
		feeStepOne = true
	}

	liqFrom, liqTo := tc.Pool.GetLiquidityAmounts(tc.DenomGiving)

	tradeData := trading.TradeData{
		MaxPrice:           tc.MaxPrice,
		Callbacks:          tc.Callbacks,
		TradeAmount:        tc.TradeAmount,
		MinimumTradeAmount: tc.MinimumTradeAmount,
		LiqFrom:            liqFrom,
		LiqTo:              liqTo,
		Fee:                tc.Pool.PoolFee,
		FeeStepOne:         feeStepOne,
	}

	tradeData.Callbacks.GetFee = getFeeAmount
	if tc.CPTrade != nil {
		tradeData.Callbacks.CPTrade = tc.CPTrade
	}

	return tradeData
}
