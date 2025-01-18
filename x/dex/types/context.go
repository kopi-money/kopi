package types

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/x/dex/constant_product"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
)

type TradeType int

const (
	TradeTypeSell = iota + 1
	TradeTypeBuy
)

type AdditionalLiquidity struct {
	addLiq  map[string]math.LegacyDec
	sizeFac map[string]math.LegacyDec
}

func (al *AdditionalLiquidity) Set(denom string, value math.LegacyDec) {
	if al.addLiq == nil {
		al.addLiq = make(map[string]math.LegacyDec)
	}

	al.addLiq[denom] = value
}

func (al *AdditionalLiquidity) Add(denom string, value math.LegacyDec) math.LegacyDec {
	addLiq, has := al.addLiq[denom]
	if has {
		value = value.Add(addLiq)
	}

	return value
}

func (al *AdditionalLiquidity) SetSizeFactor(denom string, value math.LegacyDec) {
	if al.sizeFac == nil {
		al.sizeFac = make(map[string]math.LegacyDec)
	}

	al.sizeFac[denom] = value
}

func (al *AdditionalLiquidity) GetSizeFactor(denom string) math.LegacyDec {
	sizeFactor, has := al.sizeFac[denom]
	if !has {
		return math.LegacyOneDec()
	}

	return sizeFactor
}

type TradeContext struct {
	context.Context

	Fee                    math.LegacyDec
	TradeType              TradeType
	TradeAmount            math.Int
	MaxPrice               *math.LegacyDec
	MinimumTradeAmount     *math.Int
	MaximumAvailableAmount math.Int
	AdditionalLiquidity    AdditionalLiquidity

	TradeDenomGiving    string
	TradeDenomReceiving string

	ExcludeFromDiscount bool
	ProtocolTrade       bool
	IsOrder             bool

	CoinSource      string
	CoinTarget      string
	DiscountAddress string

	CalcMaximumTradableAmount      func(TradeContext) (*math.Int, error)
	CalcTradableAmountGivenPrice   constant_product.CalculateMaximumAmount
	CalcAmountToGive               func() (math.Int, error)
	IntermediateTradeAmount        IntermediateTradeAmount
	CalcMaximumTradeAmountByWallet func() (math.Int, error)

	TradeBalances TradeBalances
	OrdersCaches  *OrdersCaches

	FlatPrice *constant_product.FlatPrice
}

func (tc *TradeContext) GetAmountGiven(result math.LegacyDec) math.Int {
	switch tc.TradeType {
	case TradeTypeSell:
		return tc.TradeAmount
	case TradeTypeBuy:
		return result.Ceil().TruncateInt()
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) GetAmountReceived(result math.LegacyDec) math.Int {
	switch tc.TradeType {
	case TradeTypeSell:
		return result.TruncateInt()
	case TradeTypeBuy:
		return tc.TradeAmount
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) DoFirstStep() bool {
	switch tc.TradeType {
	case TradeTypeSell:
		return constants.BaseCurrency != tc.TradeDenomGiving
	case TradeTypeBuy:
		return constants.BaseCurrency != tc.TradeDenomReceiving
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) DoSecondStep() bool {
	switch tc.TradeType {
	case TradeTypeSell:
		return constants.BaseCurrency != tc.TradeDenomReceiving
	case TradeTypeBuy:
		return constants.BaseCurrency != tc.TradeDenomGiving
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) FirstGiving() string {
	switch tc.TradeType {
	case TradeTypeSell:
		return tc.TradeDenomGiving
	case TradeTypeBuy:
		return constants.BaseCurrency
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) SecondGiving() string {
	switch tc.TradeType {
	case TradeTypeSell:
		return constants.BaseCurrency
	case TradeTypeBuy:
		return tc.TradeDenomGiving
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) FirstReceiving() string {
	switch tc.TradeType {
	case TradeTypeSell:
		return constants.BaseCurrency
	case TradeTypeBuy:
		return tc.TradeDenomReceiving
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) SecondReceiving() string {
	switch tc.TradeType {
	case TradeTypeSell:
		return tc.TradeDenomReceiving
	case TradeTypeBuy:
		return constants.BaseCurrency
	default:
		panic("trade type not set")
	}
}

func (tc *TradeContext) GetOrdersCaches() *OrdersCaches {
	return tc.OrdersCaches
}

func (tc *TradeContext) FullFee() math.LegacyDec {
	if tc.HasOneStep() {
		return tc.StepFee()
	}

	return tc.Fee
}

func (tc *TradeContext) HasOneStep() bool {
	return tc.TradeDenomGiving == constants.BaseCurrency || tc.TradeDenomReceiving == constants.BaseCurrency
}

func (tc *TradeContext) HasTwoSteps() bool {
	return !tc.HasOneStep()
}

func (tc *TradeContext) StepFee() math.LegacyDec {
	return tc.Fee.Quo(math.LegacyNewDec(2)) // C
}

func (tc *TradeContext) CalcStepFee(fee math.LegacyDec) math.LegacyDec {
	if tc.HasTwoSteps() {
		return fee.Quo(math.LegacyNewDec(2)) // C
	}

	return fee
}

func (tc *TradeContext) ToSell(amount math.Int) TradeContext {
	return TradeContext{
		Context:                tc.Context,
		CoinSource:             tc.CoinSource,
		CoinTarget:             tc.CoinTarget,
		TradeAmount:            amount,
		MaximumAvailableAmount: tc.MaximumAvailableAmount,
		MaxPrice:               tc.MaxPrice,
		MinimumTradeAmount:     tc.MinimumTradeAmount,
		TradeDenomGiving:       tc.TradeDenomGiving,
		TradeDenomReceiving:    tc.TradeDenomReceiving,
		ProtocolTrade:          tc.ProtocolTrade,
		TradeBalances:          tc.TradeBalances,
		Fee:                    tc.Fee,
	}
}

type IntermediateTradeAmount func(math.Int, math.Int) math.Int

func IntermediateTradeAmountReceived(_, amount math.Int) math.Int {
	return amount
}

func IntermediateTradeAmountUsed(amount, _ math.Int) math.Int {
	return amount
}

type TradeResults struct {
	Step1 TradeResult
	Step2 TradeResult

	FeePaid1 math.Int
	FeePaid2 math.Int
}

func (tr TradeResults) Get(tradeType TradeType) TradeResult {
	if tradeType == TradeTypeSell {
		return TradeResult{
			AmountIntermediate: tr.Step1.AmountReceived,
			AmountGiven:        tr.Step1.AmountGiven,
			AmountReceived:     tr.Step2.AmountReceived,
			FeeBase:            tr.FeePaid2,
			FeeOther:           tr.FeePaid1,
		}
	} else {
		return TradeResult{
			AmountIntermediate: tr.Step1.AmountGiven,
			AmountGiven:        tr.Step2.AmountGiven,
			AmountReceived:     tr.Step1.AmountReceived,
			FeeBase:            tr.FeePaid1,
			FeeOther:           tr.FeePaid2,
		}
	}
}

type TradeSimulationResult struct {
	AmountIntermediate math.Int
	AmountGiven        math.Int
	AmountReceived     math.Int
	FeeGiven           math.Int
}

type TradeResult struct {
	AmountIntermediate math.Int
	AmountGiven        math.Int
	AmountReceived     math.Int
	FeeBase            math.Int
	FeeOther           math.Int
}

func (tr TradeResult) PricePaid() (math.LegacyDec, error) {
	if !tr.AmountReceived.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("amount received not positive")
	}

	return tr.AmountGiven.ToLegacyDec().Quo(tr.AmountReceived.ToLegacyDec()), nil // C
}

type Sender interface {
	SendCoins(ctx context.Context, address sdk.AccAddress, accAddress sdk.AccAddress, coins sdk.Coins) error
}

type TradeBalances interface {
	AddTransfer(string, string, string, math.Int)
	NetBalance(string, string) math.Int
	Settle(context.Context, Sender) error
}

func plain(_, _, amount, _ math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	return amount, math.LegacyZeroDec(), nil
}

type TradeStepContext struct {
	TradeContext

	StepDenomGiving    string
	StepDenomReceiving string
	FeeDenom           string

	TradeAmount     math.Int
	ReserveFeeShare math.LegacyDec

	CalcAmountToGive    constant_product.ConstantProductTrade
	CalcAmountToReceive constant_product.ConstantProductTrade
}

// When selling: givingDenom > XKP
// When buying: XKP > receivingDenom
func (tc *TradeContext) TradeStep1(reserveFeeShare math.LegacyDec, tradeType TradeType) TradeStepContext {
	var (
		calcAmountToGive    constant_product.ConstantProductTrade
		calcAmountToReceive constant_product.ConstantProductTrade

		denomGiving    string
		denomReceiving string
	)

	switch tradeType {
	case TradeTypeSell:
		denomGiving = tc.TradeDenomGiving
		denomReceiving = constants.BaseCurrency

		calcAmountToGive = plain

		if tc.FlatPrice != nil {
			calcAmountToReceive = tc.FlatPrice.Sell
		} else {
			calcAmountToReceive = constant_product.ConstantProductTradeSell
		}

	case TradeTypeBuy:
		denomGiving = constants.BaseCurrency
		denomReceiving = tc.TradeDenomReceiving

		calcAmountToReceive = plain

		if tc.FlatPrice != nil {
			calcAmountToGive = tc.FlatPrice.Buy
		} else {
			calcAmountToGive = constant_product.ConstantProductTradeBuy
		}
	default:
		panic(fmt.Sprintf("unknown trade type: %v", tradeType))
	}

	tc.TradeType = tradeType
	return TradeStepContext{
		TradeContext:        *tc,
		StepDenomGiving:     denomGiving,
		StepDenomReceiving:  denomReceiving,
		TradeAmount:         tc.TradeAmount,
		ReserveFeeShare:     reserveFeeShare,
		CalcAmountToGive:    calcAmountToGive,
		CalcAmountToReceive: calcAmountToReceive,
	}
}

// When selling: XKP > receivingDenom
// When buying: givingDenom > XKP
func (tc *TradeContext) TradeStep2(reserveFeeShare math.LegacyDec, amount math.Int, tradeType TradeType) TradeStepContext {
	var (
		calcAmountToGive    constant_product.ConstantProductTrade
		calcAmountToReceive constant_product.ConstantProductTrade

		denomGiving    string
		denomReceiving string
	)

	switch tradeType {
	case TradeTypeSell:
		denomGiving = constants.BaseCurrency
		denomReceiving = tc.TradeDenomReceiving

		calcAmountToGive = plain
		calcAmountToReceive = constant_product.ConstantProductTradeSell
		if tc.FlatPrice != nil {
			calcAmountToReceive = tc.FlatPrice.Sell
		}

	case TradeTypeBuy:
		denomGiving = tc.TradeDenomGiving
		denomReceiving = constants.BaseCurrency

		calcAmountToReceive = plain
		calcAmountToGive = constant_product.ConstantProductTradeBuy
		if tc.FlatPrice != nil {
			calcAmountToGive = tc.FlatPrice.Buy
		}

	default:
		panic(fmt.Sprintf("unknown trade type: %v", tradeType))
	}

	tc.TradeType = tradeType
	return TradeStepContext{
		TradeContext:        *tc,
		StepDenomGiving:     denomGiving,
		StepDenomReceiving:  denomReceiving,
		TradeAmount:         amount,
		ReserveFeeShare:     reserveFeeShare,
		CalcAmountToGive:    calcAmountToGive,
		CalcAmountToReceive: calcAmountToReceive,
	}
}

func (tc *TradeContext) IsBuy() bool {
	return tc.TradeType == TradeTypeBuy
}

type TradeCalculation interface {
	// Forward is for sells, ie users give a fixed amount and receive a calculated amount
	Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int
	// Backward is for buys, ie users give receive fixed amount and give a calculated amount
	Backward(poolFrom, poolTo, result math.LegacyDec) math.Int
}
