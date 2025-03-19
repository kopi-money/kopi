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

const (
	TradeStep1 = iota + 1
	TradeStep2
)

type CutLiquidity struct {
	CutBase        math.LegacyDec
	CutOther       math.LegacyDec
	SpreadLiqBase  math.LegacyDec
	SpreadLiqOther math.LegacyDec
	VirtualBase    math.LegacyDec
	VirtualOther   math.LegacyDec

	DenomGiving    string
	DenomReceiving string
}

func (cl *CutLiquidity) GetFullFrom(denomGiving string) (math.LegacyDec, math.LegacyDec) {
	if denomGiving == constants.BaseCurrency {
		return cl.GetFullBase()
	} else {
		return cl.GetFullOther()
	}
}

func (cl *CutLiquidity) GetFullTo(denomGiving string) (math.LegacyDec, math.LegacyDec) {
	if denomGiving == constants.BaseCurrency {
		return cl.GetFullOther()
	} else {
		return cl.GetFullBase()
	}
}

func (cl *CutLiquidity) GetFullOther() (math.LegacyDec, math.LegacyDec) {
	return cl.CutOther, cl.VirtualOther
}

func (cl *CutLiquidity) GetFullOtherSummed() math.LegacyDec {
	v1, v2 := cl.GetFullOther()
	return v1.Add(v2)
}

func (cl *CutLiquidity) GetFullBase() (math.LegacyDec, math.LegacyDec) {
	return cl.CutBase, cl.VirtualBase
}

func (cl *CutLiquidity) GetFullBaseSummed() math.LegacyDec {
	v1, v2 := cl.GetFullBase()
	return v1.Add(v2)
}

func (cl *CutLiquidity) GetTradeLiquidities(denomGiving string) (math.LegacyDec, math.LegacyDec) {
	ab, vb := cl.GetFullBase()
	ao, vo := cl.GetFullOther()

	fullBase := ab.Add(vb)
	fullOther := ao.Add(vo)

	if denomGiving == constants.BaseCurrency {
		return fullBase, fullOther
	} else {
		return fullOther, fullBase
	}
}

type CutLiquidities struct {
	Step1 *CutLiquidity
	Step2 *CutLiquidity

	LiqChangeFrom math.LegacyDec
	LiqChangeTo   math.LegacyDec
}

func (cl *CutLiquidities) SizeFactor() math.LegacyDec {
	if cl.Step1 == nil || cl.Step2 == nil {
		return math.LegacyOneDec()
	}

	tradeValue1 := cl.Step1.GetFullBaseSummed()
	tradeValue2 := cl.Step2.GetFullBaseSummed()
	return tradeValue1.Sub(tradeValue2)
}

func (cl *CutLiquidities) IsZeroTrade(tradeType TradeType) bool {
	switch tradeType {
	case TradeTypeSell:
		if cl.Step1 != nil {
			ab, vb := cl.Step1.GetFullBase()
			if !ab.Add(vb).IsPositive() {
				return true
			}
		}
	case TradeTypeBuy:
		if cl.Step2 != nil {
			ab, vb := cl.Step2.GetFullBase()
			if !ab.Add(vb).IsPositive() {
				return true
			}
		}
	default:
		panic("unknown tradeType")
	}

	return false
}

func (cl *CutLiquidities) UpdateBaseRelational(tradeType TradeType, amountGiven, amountReceived math.Int) math.LegacyDec {
	if cl.Step2 == nil {
		return math.LegacyZeroDec()
	}

	amountBaseStep1Original := cl.Step1.CutBase
	if amountBaseStep1Original.IsNil() {
		amountBaseStep1Original = math.LegacyZeroDec()
	}

	var amountBaseStep1Adjusted math.LegacyDec
	if tradeType == TradeTypeSell {
		amountBaseStep1Adjusted = amountBaseStep1Original.Sub(amountReceived.ToLegacyDec())
	} else {
		amountBaseStep1Adjusted = amountBaseStep1Original.Add(amountGiven.ToLegacyDec())
	}

	return amountBaseStep1Adjusted.Sub(amountBaseStep1Original)
}

func (cl *CutLiquidities) UpdateBaseFixed(amount math.LegacyDec) {
	if cl.Step2 == nil {
		return
	}

	amountBase := cl.Step2.CutBase
	if amountBase.IsNil() {
		amountBase = math.LegacyZeroDec()
	}

	amountBase = amountBase.Add(amount)
	cl.Step2.CutBase = amountBase
}

func (cl *CutLiquidities) GetFullLiquidityGiving(denom string, tradeType TradeType) math.LegacyDec {
	if tradeType == TradeTypeSell {
		if denom != constants.BaseCurrency {
			return cl.Step1.GetFullOtherSummed()
		} else {
			return cl.Step2.GetFullBaseSummed()
		}
	} else {
		if denom != constants.BaseCurrency {
			return cl.Step2.GetFullOtherSummed()
		} else {
			return cl.Step1.GetFullBaseSummed()
		}
	}
}

func (cl *CutLiquidities) GetFullLiquidityReceiving(denom string, tradeType TradeType) math.LegacyDec {
	if tradeType == TradeTypeSell {
		if denom != constants.BaseCurrency {
			return cl.Step2.GetFullOtherSummed()
		} else {
			return cl.Step1.GetFullBaseSummed()
		}
	} else {
		if denom != constants.BaseCurrency {
			return cl.Step1.GetFullOtherSummed()
		} else {
			return cl.Step2.GetFullBaseSummed()
		}
	}
}

func (cl *CutLiquidities) GetFullBase() math.LegacyDec {
	if cl.Step1 != nil {
		return cl.Step1.GetFullBaseSummed()
	} else {
		return cl.Step2.GetFullBaseSummed()
	}
}

type TradeContext struct {
	context.Context

	Fee                    math.LegacyDec
	TradeType              TradeType
	TradeAmount            math.Int
	MaxPrice               *math.LegacyDec
	MinimumTradeAmount     *math.Int
	MaximumAvailableAmount math.Int
	CutLiquidities         CutLiquidities
	liquidityChanges       *AmountsMap

	TradeDenomGiving    string
	TradeDenomReceiving string

	ExcludeFromDiscount bool
	ProtocolTrade       bool
	IsOrder             bool

	CoinSource      string
	CoinTarget      string
	DiscountAddress string

	CalcMaximumTradableAmount           func(TradeContext) (*math.Int, error)
	CalcTradableAmountGivenPriceOneStep constant_product.CalculateMaximumAmountOneStep
	CalcTradableAmountGivenPriceTwoStep constant_product.CalculateMaximumAmountTwoStep
	CalcAmountToGive                    func() (math.Int, error)
	IntermediateTradeAmount             IntermediateTradeAmount
	CalcMaximumTradeAmountByWallet      func() (math.Int, error)

	TradeBalances TradeBalances
	OrdersCaches  *OrdersCaches

	FlatPrice *constant_product.FlatPrice
}

func (tc *TradeContext) TouchedDenom(denom string) bool {
	return tc.TradeDenomReceiving == denom || tc.TradeDenomGiving == denom
}

func (tc *TradeContext) AddLiquidityChange(denom string, amount math.Int) {
	if tc.liquidityChanges == nil {
		tc.liquidityChanges = NewAmountsMap()
	}

	tc.liquidityChanges.Add(denom, amount.ToLegacyDec())
}

func (tc *TradeContext) GetLiquidityChange(denom string) math.Int {
	if tc.liquidityChanges == nil {
		tc.liquidityChanges = NewAmountsMap()
	}

	return tc.liquidityChanges.AmountOf(denom).TruncateInt()
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

func (tsr TradeSimulationResult) PricePaid() (math.LegacyDec, error) {
	if !tsr.AmountReceived.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("amount received not positive")
	}

	return tsr.AmountGiven.ToLegacyDec().Quo(tsr.AmountReceived.ToLegacyDec()), nil
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
	*TradeContext

	StepDenomGiving    string
	StepDenomReceiving string

	TradeAmount     math.Int
	ReserveFeeShare math.LegacyDec
	CutLiquidity    *CutLiquidity

	CalcAmountToGive    constant_product.ConstantProductTrade
	CalcAmountToReceive constant_product.ConstantProductTrade

	TradeStepIndex int
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
		TradeContext:        tc,
		StepDenomGiving:     denomGiving,
		StepDenomReceiving:  denomReceiving,
		TradeAmount:         tc.TradeAmount,
		ReserveFeeShare:     reserveFeeShare,
		CalcAmountToGive:    calcAmountToGive,
		CalcAmountToReceive: calcAmountToReceive,
		CutLiquidity:        tc.CutLiquidities.Step1,
		TradeStepIndex:      TradeStep1,
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
		TradeContext:        tc,
		StepDenomGiving:     denomGiving,
		StepDenomReceiving:  denomReceiving,
		TradeAmount:         amount,
		ReserveFeeShare:     reserveFeeShare,
		CalcAmountToGive:    calcAmountToGive,
		CalcAmountToReceive: calcAmountToReceive,
		CutLiquidity:        tc.CutLiquidities.Step2,
		TradeStepIndex:      TradeStep2,
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
