package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	"github.com/kopi-money/kopi/x/dex/types"
)

var errSwitchToSell = errors.New("switch to sell")

func (k Keeper) ExecuteSell(ctx types.TradeContext) (types.TradeResult, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeSell
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumSellableAmount
	ctx.CalcTradableAmountGivenPriceOneStep = constant_product.CalculateMaximumGivingOneStep
	ctx.CalcTradableAmountGivenPriceTwoStep = constant_product.CalculateMaximumGivingTwoStep

	ctx.CalcAmountToGive = func() (math.Int, error) {
		return ctx.TradeAmount, nil
	}
	ctx.IntermediateTradeAmount = types.IntermediateTradeAmountReceived
	ctx.CalcMaximumTradeAmountByWallet = func() (math.Int, error) {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		amount := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount

		if ctx.MaximumAvailableAmount.IsNil() {
			ctx.MaximumAvailableAmount = k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount
		}

		return math.MinInt(amount, ctx.MaximumAvailableAmount), nil
	}

	result, err := k.executeTrade(&ctx)
	if err != nil {
		return types.TradeResult{}, err
	}

	tradeResult := result.Get(types.TradeTypeSell)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: ctx.CoinTarget},
			sdk.Attribute{Key: "denom_giving", Value: ctx.TradeDenomGiving},
			sdk.Attribute{Key: "denom_receiving", Value: ctx.TradeDenomReceiving},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: tradeResult.AmountIntermediate.String()},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven.String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived.String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(ctx.ProtocolTrade)},
		),
	)

	return tradeResult, nil
}

func (k Keeper) ExecuteBuy(ctx types.TradeContext) (types.TradeResult, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeBuy
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumBuyableAmount
	ctx.CalcTradableAmountGivenPriceOneStep = constant_product.CalculateMaximumReceiving
	ctx.CalcTradableAmountGivenPriceTwoStep = constant_product.CalculateMaximumReceivingTwoStep
	ctx.CalcAmountToGive = func() (math.Int, error) {
		tradeResult, err := k.SimulateSell(ctx)
		if err != nil {
			return math.Int{}, err
		}

		return tradeResult.AmountGiven, nil
	}

	ctx.CalcMaximumTradeAmountByWallet = func() (math.Int, error) {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		available := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount

		if !ctx.MaximumAvailableAmount.IsNil() {
			available = math.MinInt(available, ctx.MaximumAvailableAmount)
		}

		original := ctx.TradeAmount
		ctx.TradeAmount = available
		res, err := k.SimulateSellWithFee(ctx, ctx.Fee)
		if err != nil {
			return math.Int{}, err
		}

		ctx.TradeAmount = original
		return res.AmountReceived, nil
	}

	ctx.IntermediateTradeAmount = types.IntermediateTradeAmountUsed

	if ctx.MaximumAvailableAmount.IsNil() {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		ctx.MaximumAvailableAmount = k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount
	}

	result, err := k.executeTrade(&ctx)
	if err != nil {
		if errors.Is(err, errSwitchToSell) {
			return k.ExecuteSell(ctx)
		}

		return types.TradeResult{}, err
	}

	tradeResult := result.Get(types.TradeTypeBuy)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: ctx.CoinTarget},
			sdk.Attribute{Key: "denom_giving", Value: ctx.TradeDenomGiving},
			sdk.Attribute{Key: "denom_receiving", Value: ctx.TradeDenomReceiving},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: tradeResult.AmountIntermediate.String()},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven.String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived.String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(ctx.ProtocolTrade)},
		),
	)

	return tradeResult, nil
}

func (k Keeper) executeTrade(ctx *types.TradeContext) (types.TradeResults, error) {
	if err := k.validateTradeOptions(ctx); err != nil {
		return types.TradeResults{}, fmt.Errorf("error in trade options: %w", err)
	}

	if ctx.Fee.IsNil() {
		fee := k.GetParams(ctx).TradeFee
		ctx.Fee = k.getTradeFee(ctx, fee, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	}

	// The address executing the trade might not be the one eligible for a discount. For example, the protocol might
	// sell a user's collateral to partially repay a loan. The protocol does not receive discount when trading, but the
	// user being liquidated does.
	if ctx.DiscountAddress == "" {
		ctx.DiscountAddress = ctx.CoinTarget
	}

	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	// When the trade is touching three denoms, maxLiqBase holds the maximum amount of liquidity of those three denoms
	// expressed in the base denom
	k.PrepareCutLiquidity(ctx)
	if ctx.CutLiquidities.IsZeroTrade(ctx.TradeType) {
		return types.TradeResults{}, types.ErrZeroTrade
	}

	// When selling:
	// With the given funds and the liquidity on the DEX, we can calculate how much a user is to receive when trading.
	// In some cases though, caused by virtual liquidity, the user would receive more than there is liquidity present.
	// In those cases, the given amount is lowered if the user is okay with an incomplete trade. If not, an error is
	// returned.
	// When buying:
	// Given how much funds are in the user's wallet, the user might not get the full desired amount.

	maximumTradableAmount, err := ctx.CalcMaximumTradableAmount(*ctx)
	if err != nil {
		return types.TradeResults{}, err
	}

	if maximumTradableAmount != nil && maximumTradableAmount.LT(ctx.TradeAmount) {
		if ctx.MinimumTradeAmount != nil && maximumTradableAmount.LT(*ctx.MinimumTradeAmount) {
			return types.TradeResults{}, types.ErrNotEnoughLiquidity
		}

		ctx.TradeAmount = *maximumTradableAmount
	}

	// When selling:
	// When a maximum price is set, it is checked how much can be given to stay below the maximum price. If that amount
	// is lower than what it is intended to be given, it means trading with the intended amount would result in a higher
	// price than wanted. In that case, the trade amount is either lowered when the user accepts an incomplete trade, or
	// an error is returned.
	// When buying:
	// When a maximum price is set, it is checked how much can be received to stay below the maximum price.

	if ctx.MaxPrice != nil {
		priceAmountDec, err := k.calculateAmountGivenPrice(ctx)
		if err != nil {
			return types.TradeResults{}, err
		}

		priceAmount := priceAmountDec.TruncateInt()
		if !priceAmount.IsPositive() {
			return types.TradeResults{}, types.ErrNegativeTradeAmount
		}

		if priceAmount.LT(ctx.TradeAmount) {
			if ctx.MinimumTradeAmount != nil && !ctx.MinimumTradeAmount.IsNil() && ctx.MinimumTradeAmount.GT(priceAmount) {
				return types.TradeResults{}, types.ErrPriceTooLow
			}

			ctx.TradeAmount = priceAmount
		}
	}

	availableByWallet, err := ctx.CalcMaximumTradeAmountByWallet()
	if err != nil {
		return types.TradeResults{}, fmt.Errorf("calculating maximum trade amount by wallet: %w", err)
	}

	if availableByWallet.LT(ctx.TradeAmount) {
		if ctx.MinimumTradeAmount != nil && availableByWallet.LT(*ctx.MinimumTradeAmount) {
			return types.TradeResults{}, types.ErrNotEnoughFunds
		}

		if ctx.TradeType == types.TradeTypeBuy {
			ctx.TradeAmount = availableByWallet
			return types.TradeResults{}, errSwitchToSell
		}

		ctx.TradeAmount = availableByWallet
	}

	// If the trade amount is below a given minimum trade amount, an error is returned.
	if ctx.MinimumTradeAmount != nil && ctx.TradeAmount.LT(*ctx.MinimumTradeAmount) {
		return types.TradeResults{}, types.ErrTradeAmountTooSmall
	}

	// If the trade amount is too small, an error is returned. The reason for that is that small trade amounts are more
	// affected by rounding issues.
	if ctx.TradeAmount.LT(math.NewInt(constants.MinimumTradeSize)) {
		return types.TradeResults{}, types.ErrTradeAmountTooSmall
	}

	// First trade step from the starting currency to the base currency
	tradeStepCtx := ctx.TradeStep1(ctx.OrdersCaches.ReserveFeeShare.Get(), ctx.TradeType)
	amountUsed1, amountReceived1, feePaid1, err := k.ExecuteTradeStep(tradeStepCtx)
	if err != nil {
		return types.TradeResults{}, fmt.Errorf("could not execute trade step 1: %w", err)
	}

	tradeAmount := ctx.IntermediateTradeAmount(amountUsed1, amountReceived1)
	if !tradeAmount.IsPositive() {
		return types.TradeResults{}, types.ErrZeroTrade
	}

	if ctx.IsOrder {
		tradeAmount = k.handleOrderFee(ctx.OrdersCaches, ctx.TradeBalances, ctx.OrdersCaches.OrderFee.Get(), tradeAmount, ctx.IsBuy())
	}

	// Second trade from the base currency to the target currency
	tradeStepCtx = ctx.TradeStep2(ctx.OrdersCaches.ReserveFeeShare.Get(), tradeAmount, ctx.TradeType)
	amountUsed2, amountReceived2, feePaid2, err := k.ExecuteTradeStep(tradeStepCtx)
	if err != nil {
		return types.TradeResults{}, fmt.Errorf("could not execute trade step 2: %w", err)
	}

	if err = k.updateRatiosToBase(ctx); err != nil {
		return types.TradeResults{}, fmt.Errorf("update ratios to base: %w", err)
	}

	ctx.OrdersCaches.Clear()
	k.AddTradeAmount(ctx, ctx.CoinTarget, amountReceived1)

	tradeResult := types.TradeResults{
		Step1: types.TradeResult{
			AmountIntermediate: amountReceived1,
			AmountGiven:        amountUsed1,
			AmountReceived:     amountReceived1,
		},
		Step2: types.TradeResult{
			AmountIntermediate: amountReceived1,
			AmountGiven:        amountUsed2,
			AmountReceived:     amountReceived2,
		},
		FeePaid1: feePaid1,
		FeePaid2: feePaid2,
	}

	return tradeResult, nil
}

// ExecuteTradeStep is called twice for each trade since every trade is routed via the base currency. If a user trades
// to or from the base currency, it means in one of the two steps nothing is done. The method calculates how much the
// trading user receives of the "To" currency given his offered amount of the "From" currency. Then, the method
// iterates over the liquidity list for that denom. For each liquidity entry it is checked whether that entry can cover
// all the needed amount. If no, that entry is removed and the next one is used. Also, for each iteration, the
// user offering liquidity gets a fee. The fee is given in the "From" currency and is added as new liquidity for the
// liquidity providing user.
func (k Keeper) ExecuteTradeStep(ctx types.TradeStepContext) (math.Int, math.Int, math.Int, error) {
	accPoolTrade := ctx.OrdersCaches.AccPoolTrade.Get().String()
	accPoolReserve := ctx.OrdersCaches.AccPoolReserve.Get().String()
	accPoolFeeIncome := ctx.OrdersCaches.AccPoolFeeIncome.Get().String()

	// If a trade is from XKP to something else, the following step sends XKP to the module in trade step 1
	if ctx.StepDenomReceiving == constants.BaseCurrency && ctx.TradeDenomGiving == constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, constants.BaseCurrency, ctx.TradeAmount)
		return ctx.TradeAmount, ctx.TradeAmount, math.ZeroInt(), nil
	}

	// If a trade is from something else to XKP, the following step sends XKP to the user in trade step 2
	if ctx.StepDenomGiving == constants.BaseCurrency && ctx.TradeDenomReceiving == constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomReceiving, ctx.TradeAmount)
		return ctx.TradeAmount, ctx.TradeAmount, math.ZeroInt(), nil
	}

	poolFrom1 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomGiving)
	poolTo1 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomReceiving)
	fullFrom, fullTo := ctx.CutLiquidity.GetTradeLiquidities(ctx.StepDenomGiving)

	amountToGiveGross, feeGiving, amountToReceiveGross, feeReceiving, err := k.calculateTradeAmounts(ctx, fullFrom, fullTo, ctx.TradeAmount.ToLegacyDec(), ctx.StepFee())
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not calculate trade amounts: %w", err)
	}

	if !amountToGiveGross.IsPositive() || !amountToReceiveGross.IsPositive() {
		return math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), nil
	}

	liquidityProviders, amountToReceiveLeft, err := k.determineLiquidityProviders(ctx, amountToReceiveGross.Add(feeReceiving), ctx.StepDenomReceiving)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not send from source to dex (2): %w", err)
	}

	if amountToReceiveLeft.IsPositive() {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not fulfill trade")
	}

	if !amountToReceiveGross.IsPositive() {
		return math.Int{}, math.Int{}, math.Int{}, types.ErrZeroTrade
	}

	amountActuallyReceivedGross := amountToReceiveGross.Sub(amountToReceiveLeft)

	shareUsed := math.LegacyZeroDec()
	if amountActuallyReceivedGross.IsPositive() {
		shareUsed = amountActuallyReceivedGross.ToLegacyDec().Quo(amountToReceiveGross.ToLegacyDec()) // C
	}

	amountUsedNet := shareUsed.Mul(amountToGiveGross.ToLegacyDec()).RoundInt()

	if ctx.StepDenomGiving != constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, ctx.StepDenomGiving, amountUsedNet)
	}

	var feePaid, feeForReserve, feeForLiquidityProviders, fundsToDistribute math.Int
	var reserveFeeDenom string

	if ctx.TradeType == types.TradeTypeSell {
		feePaid, feeForReserve, feeForLiquidityProviders = manageFee(feeReceiving, ctx.ReserveFeeShare)

		reserveFeeDenom = ctx.StepDenomReceiving
		fundsToDistribute = amountToGiveGross
	} else {
		feePaid, feeForReserve, feeForLiquidityProviders = manageFee(feeGiving, ctx.ReserveFeeShare)

		reserveFeeDenom = ctx.StepDenomGiving
		fundsToDistribute = amountToGiveGross.Sub(feePaid)
	}

	if feeForLiquidityProviders.IsPositive() {
		ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolFeeIncome, reserveFeeDenom, feeForLiquidityProviders)
	}

	if err = k.distributeGivenFunds(ctx, ctx.OrdersCaches, liquidityProviders, fundsToDistribute, amountActuallyReceivedGross, ctx.StepDenomGiving); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not distribute FROM funds to liquidity providers: %w", err)
	}

	ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolReserve, reserveFeeDenom, feeForReserve)
	payoutAmount := ctx.TradeBalances.NetBalance(accPoolTrade, ctx.StepDenomReceiving)
	if ctx.StepDenomReceiving != constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomReceiving, payoutAmount)
	}

	poolFrom2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomGiving)
	poolTo2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomReceiving)
	changeFrom := poolFrom2.Sub(poolFrom1).TruncateInt()
	changeTo := poolTo2.Sub(poolTo1).TruncateInt()

	ctx.AddLiquidityChange(ctx.StepDenomGiving, changeFrom)
	ctx.AddLiquidityChange(ctx.StepDenomReceiving, changeTo)

	if ctx.TradeStepIndex == types.TradeStep1 {
		ctx.CutLiquidities.UpdateBaseRelational(ctx.TradeType, amountToGiveGross, amountToReceiveGross)
	}

	return amountToGiveGross, payoutAmount, feePaid, nil
}

func (k Keeper) updateRatiosToBase(ctx *types.TradeContext) error {
	liqBase := k.GetSpreadLiquidity(ctx, constants.BaseCurrency)
	changeBase := ctx.GetLiquidityChange(constants.BaseCurrency).ToLegacyDec()

	if ctx.HasOneStep() {
		fullBase := ctx.CutLiquidities.GetFullBase()
		for _, ratio := range k.DenomKeeper.GetAllRatios(ctx) {
			changeOther := ctx.GetLiquidityChange(ratio.Denom).ToLegacyDec()
			if err := k.updateRatioToBase(ctx, ratio, liqBase, changeBase, changeOther, fullBase); err != nil {
				return fmt.Errorf("update ratio to base (%v): %w", ratio.Denom, err)
			}
		}
	} else {
		ratioGiving, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomGiving)
		fullBase := ctx.CutLiquidities.Step1.GetFullBaseSummed()
		changeOther := ctx.GetLiquidityChange(ctx.TradeDenomGiving).ToLegacyDec()
		if err := k.updateRatioToBase(ctx, ratioGiving, liqBase, changeBase, changeOther, fullBase); err != nil {
			return fmt.Errorf("update ratio to base (step1, %v): %w", ctx.TradeDenomGiving, err)
		}

		ratioReceiving, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomReceiving)
		fullBase = ctx.CutLiquidities.Step2.GetFullBaseSummed()
		changeOther = ctx.GetLiquidityChange(ctx.TradeDenomReceiving).ToLegacyDec()
		if err := k.updateRatioToBase(ctx, ratioReceiving, liqBase, changeBase, changeOther, fullBase); err != nil {
			return fmt.Errorf("update ratio to base (step2, %v): %w", ctx.TradeDenomReceiving, err)
		}
	}

	return nil
}

func (k Keeper) updateRatioToBase(ctx *types.TradeContext, ratio denomtypes.Ratio, liqBase, changeBase, changeOther math.LegacyDec, originalTradeValue math.LegacyDec) error {
	if changeOther.IsZero() && changeBase.IsZero() {
		return nil
	}

	liqOther := k.GetSpreadLiquidity(ctx, ratio.Denom)
	extraVirtualLiquidity := k.DenomKeeper.ExtraVirtualLiquidity(ctx, ratio.Denom)

	pair, err := k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, extraVirtualLiquidity)
	if err != nil {
		return fmt.Errorf("liquidity pair %v: %w", ratio.Denom, err)
	}

	fullBase := pair.VirtualBase.Add(pair.ActualBase)
	fullOther := pair.VirtualOther.Add(pair.ActualOther)

	fullBase = fullBase.Add(pair.ExtraBase)
	fullOther = fullOther.Add(pair.ExtraOther)

	changeBase = adjustChangeToTradeValue(changeBase, originalTradeValue, fullBase)
	changeOther = adjustChangeToTradeValue(changeOther, originalTradeValue, fullBase)

	fullBase = fullBase.Add(changeBase)
	fullOther = fullOther.Add(changeOther)

	if fullBase.IsPositive() {
		newRatio := fullOther.Quo(fullBase) // C
		k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{
			Denom: ratio.Denom,
			Ratio: newRatio,
		})
	}

	return nil
}

// adjustChangeToTradeValue adjusts the change amount of a denom to that pair's liquidity level. For example, if there
// is $1k worth of XKP and 1 XKP is bought, changeRelation is 0.001, ie 0.1%. When there is a trading pair with less
// than $1k trade value, change has to be made smaller as to not be larger than 0.1% of that pair's trade value.
func adjustChangeToTradeValue(change, originalTradeValueBase, currentTradeValueBase math.LegacyDec) math.LegacyDec {
	if originalTradeValueBase.LTE(currentTradeValueBase) {
		return change
	}

	neg := change.IsNegative()
	if neg {
		change = change.Neg()
	}

	changeRelation := change.Quo(originalTradeValueBase)
	change = changeRelation.Mul(currentTradeValueBase)

	if neg {
		change = change.Neg()
	}

	return change
}

func (k Keeper) calculateTradeAmounts(ctx types.TradeStepContext, poolFrom, poolTo, tradeAmount, fee math.LegacyDec) (math.Int, math.Int, math.Int, math.Int, error) {
	amountToGive, feeGiving, err := ctx.CalcAmountToGive(poolFrom, poolTo, tradeAmount, fee)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, err
	}

	if amountToGive.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), nil
	}

	amountToReceive, feeReceiving, err := ctx.CalcAmountToReceive(poolFrom, poolTo, tradeAmount, fee)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, err
	}

	if amountToReceive.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), nil
	}

	return amountToGive.Ceil().TruncateInt(),
		feeGiving.Ceil().TruncateInt(),
		amountToReceive.TruncateInt(),
		feeReceiving.TruncateInt(), nil
}

func (k Keeper) handleOrderFee(ordersCaches *types.OrdersCaches, tradeBalances types.TradeBalances, orderFee math.LegacyDec, amount math.Int, isBuy bool) math.Int {
	var feeAmount math.Int

	if isBuy {
		feeAmount = amount.ToLegacyDec().Quo(math.LegacyOneDec().Sub(orderFee)).Sub(amount.ToLegacyDec()).TruncateInt() // C
		amount = amount.Add(feeAmount)
	} else {
		feeAmount = amount.ToLegacyDec().Mul(orderFee).TruncateInt()
		amount = amount.Sub(feeAmount)
	}

	tradeBalances.AddTransfer(
		ordersCaches.AccPoolTrade.Get().String(),
		ordersCaches.AccPoolReserve.Get().String(),
		constants.BaseCurrency, feeAmount,
	)

	return amount
}

func (k Keeper) calculateAmountGivenPrice(ctx *types.TradeContext) (math.LegacyDec, error) {
	maxPrice := ctx.MaxPrice.Mul(math.LegacyOneDec().Sub(ctx.Fee))
	if ctx.HasTwoSteps() {
		X := ctx.CutLiquidities.Step1.GetFullOtherSummed()
		T1 := ctx.CutLiquidities.Step1.GetFullBaseSummed()
		T2 := ctx.CutLiquidities.Step2.GetFullBaseSummed()
		Y := ctx.CutLiquidities.Step2.GetFullOtherSummed()

		if ctx.TradeType == types.TradeTypeBuy {
			X, Y = Y, X
			T1, T2 = T2, T1
		}

		return ctx.CalcTradableAmountGivenPriceTwoStep(X, T1, T2, Y, maxPrice)
	} else {
		liqFrom := ctx.CutLiquidities.GetFullLiquidityGiving(ctx.TradeDenomGiving, ctx.TradeType)
		liqTo := ctx.CutLiquidities.GetFullLiquidityReceiving(ctx.TradeDenomReceiving, ctx.TradeType)
		return ctx.CalcTradableAmountGivenPriceOneStep(liqFrom, liqTo, maxPrice)
	}
}

// SimulateTradeForReserve is used when calculating the profitability of a mint/burn trade. When trading, the reserve
// has to pay the trade fee. However, part of it will be paid out to itself. Thus, when estimating the profitability of
// a trade, that part of the fee is removed.
func (k Keeper) SimulateTradeForReserve(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	reserveShare := k.GetParams(ctx).ReserveShare
	fee := k.getTradeFee(ctx, ctx.Fee, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	fee = fee.Mul(math.LegacyOneDec().Sub(reserveShare))
	return k.SimulateWithFee(ctx, fee, constant_product.ConstantProductTradeSell, applySellFee)
}

// CalculateMaximumSellableAmount calculates the maximum sellable amount for a given trading pair while routing the
// trade via the base currency. First, the tradable amount between the base currency and the "to" currency is
// calculated. In the second step, the tradable amount from the "from" currency to the base currency is calculated. The
// previously calculated maximum tradable amount is given to that function to cover cases where the size bottleneck is
// in the second trading step.
func (k Keeper) CalculateMaximumSellableAmount(ctx types.TradeContext) (*math.Int, error) {
	var max1, max2 *math.Int
	if ctx.TradeDenomReceiving != constants.BaseCurrency && ctx.CutLiquidities.Step2 != nil {
		max2 = k.CalculateSingleSellableAmount(ctx.CutLiquidities.Step2, constants.BaseCurrency, nil)

		//if max2 != nil {
		//	s := ctx.CutLiquidities.SizeFactor().Mul(max2.ToLegacyDec()).TruncateInt()
		//	max2 = &s
		//}
	}

	if max2 != nil && max2.IsZero() {
		zeroInt := math.ZeroInt()
		return &zeroInt, nil
	}

	if ctx.TradeDenomGiving != constants.BaseCurrency && ctx.CutLiquidities.Step1 != nil {
		max1 = k.CalculateSingleSellableAmount(ctx.CutLiquidities.Step1, ctx.TradeDenomGiving, max2)
	} else {
		max1 = max2
	}

	return max1, nil
}

// CalculateSingleSellableAmount calculates the maximum trading amount for a given trading pair, i.e. how much of
// denomFrom can be given at maximum. When there is no virtual liquidity, the tradable amount is infinity, thus the
// return amount is nil.
func (k Keeper) CalculateSingleSellableAmount(cutLiquidity *types.CutLiquidity, denomFrom string, maximumActual *math.Int) *math.Int {
	actualFrom, virtualFrom := cutLiquidity.GetFullFrom(denomFrom)
	actualTo, virtualTo := cutLiquidity.GetFullTo(denomFrom)

	if maximumActual != nil {
		maximumActualDec := maximumActual.ToLegacyDec()
		if maximumActualDec.LT(actualTo) {
			virtualTo = actualTo.Add(virtualTo).Sub(maximumActualDec)
			actualTo = maximumActualDec
		}
	}

	if virtualTo.IsNil() || !virtualTo.IsPositive() {
		return nil
	}

	return constant_product.CalculateSingleMaximumSellableAmount(actualFrom, virtualFrom, actualTo, virtualTo)
}

// CalculateMaximumBuyableAmount...
func (k Keeper) CalculateMaximumBuyableAmount(ctx types.TradeContext) (*math.Int, error) {
	orderFee := ctx.OrdersCaches.OrderFee.Get()

	if ctx.HasOneStep() {
		var cutLiquidity *types.CutLiquidity
		if ctx.TradeDenomGiving == constants.BaseCurrency {
			cutLiquidity = ctx.CutLiquidities.Step1
		} else {
			cutLiquidity = ctx.CutLiquidities.Step2
		}

		maximum := k.CalculateSingleBuyableAmount(ctx.OrdersCaches, cutLiquidity, ctx.TradeDenomGiving, ctx.TradeDenomReceiving)
		maximum = subtractOrderFee(maximum, orderFee, ctx.IsOrder)
		return &maximum, nil
	}

	maxBase := k.CalculateSingleBuyableAmount(ctx.OrdersCaches, ctx.CutLiquidities.Step1, ctx.TradeDenomGiving, ctx.TradeDenomReceiving)
	maxBase = subtractOrderFee(maxBase, orderFee, ctx.IsOrder)

	maximum, _, err := calculateSingleTrade(constants.BaseCurrency, ctx.TradeDenomReceiving, maxBase.ToLegacyDec(), ctx.StepFee(), ctx.CutLiquidities.Step2, constant_product.ConstantProductTradeSell)
	if err != nil {
		return nil, err
	}

	available := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving)
	maximumInt := math.LegacyMinDec(maximum, available).TruncateInt()

	return &maximumInt, nil
}

func subtractOrderFee(amount math.Int, orderFee math.LegacyDec, isOrder bool) math.Int {
	if !isOrder {
		return amount
	}

	feeAmount := amount.ToLegacyDec().Quo(math.LegacyOneDec().Sub(orderFee)).Sub(amount.ToLegacyDec()).TruncateInt() // C
	return amount.Sub(feeAmount)
}

func (k Keeper) CalculateSingleBuyableAmount(ordersCaches *types.OrdersCaches, cutLiquidity *types.CutLiquidity, denomGiving, demomReceiving string) math.Int {
	actualTo := ordersCaches.LiquidityPool.Get().AmountOf(demomReceiving)
	_, virtualTo := cutLiquidity.GetFullTo(denomGiving)

	return CalculateSingleMaximumBuyableAmount(actualTo, virtualTo)
}

func CalculateSingleMaximumBuyableAmount(actualTo, virtualTo math.LegacyDec) math.Int {
	var maximum math.Int
	if virtualTo.IsNil() || virtualTo.IsZero() {
		maximum = actualTo.Sub(math.LegacyOneDec()).TruncateInt()
	} else {
		maximum = actualTo.TruncateInt()
	}

	return maximum
}

func (k Keeper) addProviderFee(ctx context.Context, amount, tradeFee math.LegacyDec) math.LegacyDec {
	feeShareReserve := k.GetReserveFeeShare(ctx)
	feeShareProvider := math.LegacyOneDec().Sub(feeShareReserve)
	feeProvider := tradeFee.Mul(feeShareProvider)

	return amount.Quo(math.LegacyOneDec().Sub(feeProvider)) // C
}

// manageFee is called each time liquidity is used for a trade. amount indicates how much is traded right now, the
// address corresponds to the user whose liquidity is used right now.
func manageFee(feeAmount math.Int, reserveFeeShare math.LegacyDec) (math.Int, math.Int, math.Int) {
	feeForReserve := feeAmount.ToLegacyDec().Mul(reserveFeeShare).RoundInt()
	feeForLiquidityProviders := feeAmount.Sub(feeForReserve)
	return feeAmount, feeForReserve, feeForLiquidityProviders
}

func (k Keeper) SimulateSell(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	fee := k.getTradeFee(ctx, ctx.Fee, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	return k.SimulateSellWithFee(ctx, fee)
}

func (k Keeper) SimulateSellWithFee(ctx types.TradeContext, fee math.LegacyDec) (types.TradeSimulationResult, error) {
	ctx.TradeType = types.TradeTypeSell
	return k.SimulateWithFee(ctx, fee, constant_product.ConstantProductTradeSell, applySellFee)
}

func (k Keeper) SimulateBuy(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	fee := k.getTradeFee(ctx, ctx.Fee, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	return k.SimulateBuyWithFee(ctx, fee)
}

func (k Keeper) SimulateBuyWithFee(ctx types.TradeContext, fee math.LegacyDec) (types.TradeSimulationResult, error) {
	ctx.TradeType = types.TradeTypeBuy
	return k.SimulateWithFee(ctx, fee, constant_product.ConstantProductTradeBuy, applyBuyFee)
}

func (k Keeper) SimulateWithFee(ctx types.TradeContext, fee math.LegacyDec, cpTrade constant_product.ConstantProductTrade, applyFee applyFee) (types.TradeSimulationResult, error) {
	if ctx.TradeAmount.IsZero() {
		return types.TradeSimulationResult{
			AmountIntermediate: math.ZeroInt(),
			AmountGiven:        math.ZeroInt(),
			AmountReceived:     math.ZeroInt(),
			FeeGiven:           math.ZeroInt(),
		}, nil
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomGiving) || !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomReceiving) {
		return types.TradeSimulationResult{}, denomtypes.ErrInvalidDexAsset
	}

	ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	k.PrepareCutLiquidity(&ctx)

	var (
		amountIntermediate math.LegacyDec
		result             math.LegacyDec
		err                error
	)

	if ctx.DoFirstStep() {
		amountIntermediate, _, err = calculateSingleTrade(ctx.FirstGiving(), ctx.FirstReceiving(), ctx.TradeAmount.ToLegacyDec(), ctx.CalcStepFee(fee), ctx.CutLiquidities.Step1, cpTrade)
		if err != nil {
			return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
		}

		if ctx.HasTwoSteps() {
			amountIntermediateInt := amountIntermediate.TruncateInt()
			var change math.LegacyDec
			if ctx.TradeType == types.TradeTypeSell {
				change = ctx.CutLiquidities.UpdateBaseRelational(ctx.TradeType, ctx.TradeAmount, amountIntermediateInt)
			} else {
				change = ctx.CutLiquidities.UpdateBaseRelational(ctx.TradeType, amountIntermediateInt, ctx.TradeAmount)
			}

			defer ctx.CutLiquidities.UpdateBaseFixed(change.Neg())
		}
	} else {
		amountIntermediate = ctx.TradeAmount.ToLegacyDec()
	}

	if ctx.DoSecondStep() {
		result, _, err = calculateSingleTrade(ctx.SecondGiving(), ctx.SecondReceiving(), amountIntermediate, ctx.CalcStepFee(fee), ctx.CutLiquidities.Step2, cpTrade)
		if err != nil {
			return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
		}
	} else {
		result = amountIntermediate
	}

	amountFee := result.Mul(fee)
	result = applyFee(result, amountFee)

	return types.TradeSimulationResult{
		AmountIntermediate: amountIntermediate.TruncateInt(),
		AmountGiven:        ctx.GetAmountGiven(result),
		AmountReceived:     ctx.GetAmountReceived(result),
		FeeGiven:           amountFee.TruncateInt(),
	}, nil
}

// CalculateSingleSell is used when simulating a sell. Since the sell is not executed, i.e. no liquidity is changed,
// this method does not need to iterate over the liquidity list but can simply calculate everything.
func CalculateSingleSell(denomGiving, denomReceiving string, offer, fee math.LegacyDec, cutLiquidity *types.CutLiquidity) (math.LegacyDec, math.LegacyDec, error) {
	return calculateSingleTrade(denomGiving, denomReceiving, offer, fee, cutLiquidity, constant_product.ConstantProductTradeSell)
}

func CalculateSingleBuy(denomGiving, denomReceiving string, requested, fee math.LegacyDec, cutLiquidity *types.CutLiquidity) (math.LegacyDec, math.LegacyDec, error) {
	return calculateSingleTrade(denomGiving, denomReceiving, requested, fee, cutLiquidity, constant_product.ConstantProductTradeBuy)
}

func calculateSingleTrade(denomGiving, denomReceiving string, offer, fee math.LegacyDec, cutLiquidity *types.CutLiquidity, cpTrade constant_product.ConstantProductTrade) (math.LegacyDec, math.LegacyDec, error) {
	if denomGiving == denomReceiving {
		return offer, math.LegacyZeroDec(), nil
	}

	poolFrom, poolTo := cutLiquidity.GetTradeLiquidities(denomGiving)
	amount, feeAmount, err := cpTrade(poolFrom, poolTo, offer, fee)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, err
	}

	return amount, feeAmount, nil
}

func (k Keeper) validateTradeOptions(ctx *types.TradeContext) error {
	if ctx.TradeBalances == nil {
		return fmt.Errorf("trade balances not set")
	}

	if ctx.CoinSource == "" {
		return types.ErrNoCoinSourceGiven
	}

	if ctx.CoinTarget == "" {
		return types.ErrNoCoinTargetGiven
	}

	if ctx.TradeDenomGiving == ctx.TradeDenomReceiving {
		return types.ErrSameDenom
	}

	if ctx.TradeAmount.IsZero() {
		return types.ErrZeroAmount
	}

	if ctx.TradeAmount.IsNegative() {
		return types.ErrNegativeAmount
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomGiving) {
		return denomtypes.ErrInvalidDexAsset
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomReceiving) {
		return denomtypes.ErrInvalidDexAsset
	}

	return nil
}

type applyFee func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func applySellFee(amount, amountFee math.LegacyDec) math.LegacyDec {
	return amount.Sub(amountFee)
}

func applyBuyFee(amount, amountFee math.LegacyDec) math.LegacyDec {
	return amount.Add(amountFee)
}
