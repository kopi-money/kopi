package trading

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
)

type MaxPriceData struct {
	MaxPrice    math.LegacyDec
	FeeIncluded bool
}

type TradeData struct {
	Callbacks

	MaxPrice           *MaxPriceData
	TradeAmount        math.Int
	MinimumTradeAmount *math.Int

	LiqFrom Liquidity
	LiqTo   Liquidity
	Fee     math.LegacyDec

	FeeStepOne bool
}

type TradeResult struct {
	TradeAmount         math.Int
	amountGivenGross    math.LegacyDec
	amountGivenNet      math.LegacyDec
	amountReceivedGross math.LegacyDec
	amountReceivedNet   math.LegacyDec
	feeDenomGiving      math.LegacyDec
	feeDenomReceiving   math.LegacyDec
}

func (tr TradeResult) AmountGiven() math.Int {
	if tr.amountGivenGross.IsNil() {
		return math.Int{}
	}

	return tr.amountGivenGross.Ceil().TruncateInt()
}

func (tr TradeResult) AmountGivenNet() math.Int {
	return tr.AmountGiven().Sub(tr.FeeGiving())
}

func (tr TradeResult) AmountReceived() math.Int {
	if tr.amountReceivedNet.IsNil() {
		return math.Int{}
	}

	return tr.amountReceivedNet.TruncateInt()
}

func (tr TradeResult) AmountReceivedGross() math.Int {
	return tr.AmountReceived().Add(tr.FeeReceiving())
}

func (tr TradeResult) FeeGiving() math.Int {
	return tr.feeDenomGiving.Ceil().TruncateInt()
}

func (tr TradeResult) FeeReceiving() math.Int {
	return tr.feeDenomReceiving.TruncateInt()
}

func (tr TradeResult) PricePaidExact() (math.LegacyDec, error) {
	if !tr.amountGivenNet.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("amount received not positive")
	}

	return tr.amountGivenGross.Quo(tr.amountReceivedNet), nil // C
}

func (tr TradeResult) PricePaidRounded() (math.LegacyDec, error) {
	if !tr.amountGivenNet.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("amount received not positive")
	}

	return tr.amountGivenGross.TruncateDec().Quo(tr.amountReceivedNet.Ceil()), nil // C
}

type Liquidity struct {
	Actual  math.LegacyDec
	Virtual math.LegacyDec
}

func (l Liquidity) Full() math.LegacyDec {
	if l.Virtual.IsNil() {
		return l.Actual
	}

	return l.Actual.Add(l.Virtual)
}

func (l Liquidity) FullNormal() math.LegacyDec {
	q := math.LegacyNewDec(1_000000)
	return l.Full().Quo(q)
}

func ParseAmount(amountStr string) (math.Int, error) {
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.ReplaceAll(amountStr, "_", "")

	amountInt, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, fmt.Errorf("invalid amount string: '%v'", amountStr)
	}

	if amountInt.LT(math.ZeroInt()) {
		return math.Int{}, ErrTradeAmountNotPositive
	}

	return amountInt, nil
}

func ParseMinimumTradeAmount(amountStr string) (*math.Int, error) {
	if amountStr == "" {
		return nil, nil
	}

	minimumTradeAmount, err := ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}

	return &minimumTradeAmount, nil
}

type getAmounts func(_, _, _, _, _, _ math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec)

func getAmountsGivingSell(amountBeforeFee, amountAfterFee, amountFee, _, _, _ math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	return amountBeforeFee, amountAfterFee, amountFee
}

func getAmountsGivingBuy(_, _, _, amountBeforeFee, amountAfterFee, amountFee math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	return amountAfterFee, amountBeforeFee, amountFee
}

func getAmountsReceivingSell(_, _, _, amountBeforeFee, amountAfterFee, amountFee math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	return amountBeforeFee, amountAfterFee, amountFee
}

func getAmountsReceivingBuy(amountBeforeFee, amountAfterFee, amountFee, _, _, _ math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	return amountAfterFee, amountBeforeFee, amountFee
}

type applyFee func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func applySellFee(amount, fee math.LegacyDec) math.LegacyDec {
	return amount.Sub(fee)
}

func applyBuyFee(amount, fee math.LegacyDec) math.LegacyDec {
	return amount.Add(fee)
}

type GetTradeDenom func(string, string) string

func getTradeDenomSell(from, _ string) string {
	return from
}

func getTradeDenomBuy(_, to string) string {
	return to
}

type Callbacks struct {
	calculateFee                CalculateFee
	CPTrade                     ConstantProductTrade
	GetFee                      GetFeeAmount
	calculateMaximumByLiquidity CalculateMaximumAmountByLiquidity
	calculateMaximumByPrice     CalculateMaximumAmountByPrice
	IsSell                      func() bool
	applyFee                    applyFee
	GetTradeDenom               GetTradeDenom

	amountsGiving    getAmounts
	amountsReceiving getAmounts
}

func SellCallbacks() Callbacks {
	return Callbacks{
		calculateFee:                CalculateSellFee,
		CPTrade:                     ConstantProductTradeSell,
		GetFee:                      GetSellFee,
		calculateMaximumByLiquidity: CalculateMaximumSellableByLiquidity,
		calculateMaximumByPrice:     CalculateMaximumSellableByPrice,
		IsSell:                      func() bool { return true },
		amountsGiving:               getAmountsGivingSell,
		amountsReceiving:            getAmountsReceivingSell,
		applyFee:                    applySellFee,
		GetTradeDenom:               getTradeDenomSell,
	}
}

func BuyCallbacks() Callbacks {
	return Callbacks{
		calculateFee:                CalculateBuyFee,
		CPTrade:                     ConstantProductTradeBuy,
		GetFee:                      GetBuyFee,
		calculateMaximumByLiquidity: CalculateMaximumBuyableByLiquidity,
		calculateMaximumByPrice:     CalculateMaximumBuyableByPrice,
		IsSell:                      func() bool { return false },
		amountsGiving:               getAmountsGivingBuy,
		amountsReceiving:            getAmountsReceivingBuy,
		applyFee:                    applyBuyFee,
		GetTradeDenom:               getTradeDenomBuy,
	}
}
