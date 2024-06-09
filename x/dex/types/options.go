package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
)

type TradeOptions struct {
	GivenAmount math.Int
	MaxPrice    *math.LegacyDec

	TradeDenomStart string
	TradeDenomEnd   string

	AllowIncomplete     bool
	ExcludeFromDiscount bool
	ProtocolTrade       bool

	CoinSource      sdk.AccAddress
	CoinTarget      sdk.AccAddress
	DiscountAddress sdk.AccAddress

	TradeCalculation TradeCalculation
	OrdersCaches     *OrdersCaches
}

type TradeStepOptions struct {
	TradeOptions

	StepDenomFrom string
	StepDenomTo   string

	Amount   math.Int
	TradeFee math.LegacyDec
}

func (to TradeOptions) TradeToBase(tradeFee math.LegacyDec) TradeStepOptions {
	return TradeStepOptions{
		TradeOptions:  to,
		StepDenomFrom: to.TradeDenomStart,
		StepDenomTo:   utils.BaseCurrency,
		Amount:        to.GivenAmount,
		TradeFee:      tradeFee,
	}
}

func (to TradeOptions) TradeToTarget(tradeFee math.LegacyDec, amount math.Int) TradeStepOptions {
	return TradeStepOptions{
		TradeOptions:  to,
		StepDenomFrom: utils.BaseCurrency,
		StepDenomTo:   to.TradeDenomEnd,
		Amount:        amount,
		TradeFee:      tradeFee,
	}
}

type TradeCalculation interface {
	Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int
	Backward(poolFrom, poolTo, result math.LegacyDec) math.Int
}
