package trading

import (
	"strings"

	"cosmossdk.io/math"
)

func ParseMaxPrice(maxPriceString string, feeIncluded bool) (*MaxPriceData, error) {
	if maxPriceString == "" {
		return nil, nil
	}

	maxPriceString = strings.ReplaceAll(maxPriceString, ",", "")
	maxPrice, err := math.LegacyNewDecFromStr(maxPriceString)
	if err != nil {
		return nil, ErrInvalidMaxPriceFormat
	}

	if !maxPrice.IsPositive() {
		return nil, ErrMaxPriceNotPositive
	}

	return &MaxPriceData{
		MaxPrice:    maxPrice,
		FeeIncluded: feeIncluded,
	}, nil
}

type AdjustMaxPrice func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func DecreaseMaxPrice(maxPrice, tradeFee math.LegacyDec) math.LegacyDec {
	return maxPrice.Mul(math.LegacyOneDec().Sub(tradeFee))
}

func KeepMaxPrice(maxPrice, _ math.LegacyDec) math.LegacyDec {
	return maxPrice
}

func HandleMaxPrice(tradeData TradeData, adjustMaxPrice AdjustMaxPrice) (math.Int, error) {
	if tradeData.MaxPrice == nil {
		return tradeData.TradeAmount, nil
	}

	if tradeData.MaxPrice.FeeIncluded {
		adjustMaxPrice = KeepMaxPrice
	}

	priceTradeAmount, err := calculateMaxAmount(tradeData.LiqFrom, tradeData.LiqTo, tradeData.Fee, tradeData.MaxPrice.MaxPrice, adjustMaxPrice, tradeData.calculateMaximumByPrice)
	if err != nil {
		return math.Int{}, err
	}

	if priceTradeAmount.LT(tradeData.TradeAmount) {
		if tradeData.MinimumTradeAmount != nil && !tradeData.MinimumTradeAmount.IsNil() && tradeData.MinimumTradeAmount.GT(priceTradeAmount) {
			return math.Int{}, ErrMarketPriceTooHigh
		}

		tradeData.TradeAmount = priceTradeAmount
	}

	if !tradeData.TradeAmount.IsPositive() {
		return math.Int{}, ErrMarketPriceTooHigh
	}

	if tradeData.MinimumTradeAmount != nil && tradeData.TradeAmount.LT(*tradeData.MinimumTradeAmount) {
		return math.Int{}, ErrPriceTooLow
	}

	return tradeData.TradeAmount, nil
}

func calculateMaxAmount(liqFrom, liqTo Liquidity, feeFactor math.LegacyDec, maxPrice math.LegacyDec, adjustMaxPrice AdjustMaxPrice, calculate CalculateMaximumAmountByPrice) (math.Int, error) {
	maxPrice = adjustMaxPrice(maxPrice, feeFactor)

	priceTradeAmount := calculate(liqFrom.Full(), liqTo.Full(), maxPrice)
	return priceTradeAmount.Ceil().TruncateInt(), nil
}

type CalculateFee func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func CalculateSellFee(feeFactor math.LegacyDec, tradeAmount math.LegacyDec) math.LegacyDec {
	return tradeAmount.Mul(feeFactor)
}

func CalculateBuyFee(feeFactor math.LegacyDec, tradeAmount math.LegacyDec) math.LegacyDec {
	tradeAmountGross := tradeAmount.Quo(math.LegacyOneDec().Sub(feeFactor))
	return tradeAmountGross.Sub(tradeAmount)
}

type GetFeeAmount func(result TradeResult) math.Int

func GetSellFee(result TradeResult) math.Int {
	return result.FeeReceiving()
}

func GetBuyFee(result TradeResult) math.Int {
	return result.FeeGiving()
}
