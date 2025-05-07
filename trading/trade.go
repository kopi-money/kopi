package trading

import (
	"cosmossdk.io/math"
)

const MinimumTradeAmount = 1_000

func Trade(tradeData TradeData) (TradeResult, error) {
	amount1BeforeFee := tradeData.TradeAmount.ToLegacyDec()

	// If the trade is to sell a kCoin, the trade fee is subtracted from the amount to be sold.
	amount1AfterFee := amount1BeforeFee
	feeAmount1 := math.LegacyZeroDec()
	if tradeData.FeeStepOne {
		feeAmount1 = tradeData.calculateFee(tradeData.Fee, amount1BeforeFee)
		amount1AfterFee = tradeData.applyFee(amount1BeforeFee, feeAmount1)
	}

	if amount1AfterFee.LT(math.LegacyNewDec(MinimumTradeAmount)) {
		return TradeResult{}, ErrTradeAmountTooSmall
	}

	// If the trade is to sell a factory token, i.e. to buy a kCoin, the trade fee is subtracted from the amount
	// to be received.
	amount2BeforeFee, err := tradeData.CPTrade(tradeData.LiqFrom.Full(), tradeData.LiqTo.Full(), amount1AfterFee)
	if err != nil {
		return TradeResult{}, err
	}

	amount2AfterFee := amount2BeforeFee
	feeAmount2 := math.LegacyZeroDec()
	if !tradeData.FeeStepOne {
		feeAmount2 = tradeData.calculateFee(tradeData.Fee, amount2BeforeFee)
		amount2AfterFee = tradeData.applyFee(amount2BeforeFee, feeAmount2)
	}

	amountGivenGross, amountGivenNet, feeDenomGiven := tradeData.amountsGiving(
		amount1BeforeFee, amount1AfterFee, feeAmount1,
		amount2BeforeFee, amount2AfterFee, feeAmount2,
	)

	amountReceivedGross, amountReceivedNet, feeDenomReceiving := tradeData.amountsReceiving(
		amount1BeforeFee, amount1AfterFee, feeAmount1,
		amount2BeforeFee, amount2AfterFee, feeAmount2,
	)

	if !amountReceivedNet.TruncateInt().IsPositive() || !amountReceivedGross.TruncateInt().IsPositive() {
		return TradeResult{}, ErrTradeAmountTooSmall
	}

	return TradeResult{
		TradeAmount:         tradeData.TradeAmount,
		amountGivenGross:    amountGivenGross,
		amountGivenNet:      amountGivenNet,
		amountReceivedGross: amountReceivedGross,
		amountReceivedNet:   amountReceivedNet,
		feeDenomGiving:      feeDenomGiven,
		feeDenomReceiving:   feeDenomReceiving,
	}, nil
}
