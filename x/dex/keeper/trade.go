package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) ExecuteSell(ctx types.TradeContext) (types.TradeResult, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeSell
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumSellableAmount
	ctx.CalcTradableAmountGivenPrice = constant_product.CalculateMaximumGiving
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

	return result.Get(types.TradeTypeSell), nil
}

func (k Keeper) ExecuteBuy(ctx types.TradeContext) (types.TradeResult, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeBuy
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumBuyableAmount
	ctx.CalcTradableAmountGivenPrice = constant_product.CalculateMaximumReceiving
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
		return types.TradeResult{}, err
	}

	return result.Get(types.TradeTypeBuy), nil
}

func (k Keeper) executeTrade(ctx *types.TradeContext) (types.TradeResults, error) {
	// return types.TradeResults{}, types.ErrTradeAmountTooSmall

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
	if err := k.PrepareCutLiquidity(ctx); err != nil {
		return types.TradeResults{}, fmt.Errorf("preparing cutliquidity: %w", err)
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

	var ratioFrom, ratioTo, liqFrom, liqTo math.LegacyDec
	if ctx.HasTwoSteps() {
		liqFrom = ctx.CutLiquidity.GetFull(ctx.TradeDenomGiving)
		liqTo = ctx.CutLiquidity.GetFull(ctx.TradeDenomReceiving)

		rf, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomGiving)
		rt, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomReceiving)
		ratioFrom, ratioTo = rf.Ratio, rt.Ratio
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

	if ctx.HasTwoSteps() {
		amountGiven := tradeResult.Get(ctx.TradeType).AmountGiven
		amountReceived := tradeResult.Get(ctx.TradeType).AmountReceived
		k.updatePairRatios(ctx, ratioFrom, ratioTo, liqFrom, liqTo, amountGiven, amountReceived)
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
	fullFrom, fullTo := GetTradeLiquidities(ctx.StepDenomGiving, ctx.StepDenomReceiving, ctx.CutLiquidity)

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
		receiveFactor := amountToGiveGross.ToLegacyDec().Quo(amountToReceiveGross.ToLegacyDec()) // C
		k.distributeSellFee(ctx, liquidityProviders, feeForLiquidityProviders, receiveFactor, ctx.StepDenomReceiving)

		reserveFeeDenom = ctx.StepDenomReceiving
		fundsToDistribute = amountToGiveGross
	} else {
		feePaid, feeForReserve, _ = manageFee(feeGiving, ctx.ReserveFeeShare)
		reserveFeeDenom = ctx.StepDenomGiving
		fundsToDistribute = amountToGiveGross.Sub(feeForReserve)
	}

	if err = k.distributeGivenFunds(ctx, ctx.OrdersCaches, liquidityProviders, fundsToDistribute, ctx.StepDenomGiving); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not distribute FROM funds to liquidity providers: %w", err)
	}

	ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolReserve, reserveFeeDenom, feeForReserve)
	payoutAmount := ctx.TradeBalances.NetBalance(accPoolTrade, ctx.StepDenomReceiving)
	if ctx.StepDenomReceiving != constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomReceiving, payoutAmount)
	}

	poolFrom2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomGiving)
	poolTo2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomReceiving)
	changeFrom := poolFrom2.Sub(poolFrom1)
	changeTo := poolTo2.Sub(poolTo1)

	ctx.CutLiquidity.UpdateBase(ctx.TradeType, amountToGiveGross, amountToReceiveGross)
	k.updateRatios(ctx, ctx.CutLiquidity.BaseValue, changeFrom, changeTo)

	return amountToGiveGross, payoutAmount, feePaid, nil
}

func (k Keeper) updateRatios(ctx types.TradeStepContext, tradeValueBase math.LegacyDec, changeFrom, changeTo math.Int) {
	var (
		baseChange  math.Int
		otherChange math.Int
		otherDenom  string
	)

	if ctx.StepDenomGiving != constants.BaseCurrency {
		baseChange = changeTo
		otherChange = changeFrom
		otherDenom = ctx.StepDenomGiving
	}

	if ctx.StepDenomReceiving != constants.BaseCurrency {
		baseChange = changeFrom
		otherChange = changeTo
		otherDenom = ctx.StepDenomReceiving
	}

	k.updateRatiosToBase(ctx, tradeValueBase, otherDenom, baseChange, otherChange)
}

func (k Keeper) updateRatiosToBase(ctx types.TradeStepContext, tradeValueBase math.LegacyDec, otherDenom string, baseChange, otherChange math.Int) {
	fullBase := tradeValueBase.Add(baseChange.ToLegacyDec())

	for _, ratio := range k.DenomKeeper.GetAllRatios(ctx) {
		if ctx.HasTwoSteps() && (ratio.Denom != ctx.TradeDenomGiving || ratio.Denom == ctx.TradeDenomReceiving) {
			continue
		}

		fullOther := tradeValueBase.Mul(ratio.Ratio)
		if ratio.Denom == otherDenom {
			fullOther = fullOther.Add(otherChange.ToLegacyDec())
		}

		if fullBase.IsPositive() {
			newRatio := fullOther.Quo(fullBase) // C
			k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{
				Denom: ratio.Denom,
				Ratio: newRatio,
			})
		}
	}
}

func (k Keeper) updatePairRatios(ctx *types.TradeContext, ratioFrom, ratioTo, fullFrom, fullTo math.LegacyDec, amountGiven, amountReceived math.Int) {
	fullFrom = fullFrom.Add(amountGiven.ToLegacyDec())
	fullTo = fullTo.Sub(amountReceived.ToLegacyDec())

	newRatio := fullTo.Quo(fullFrom)
	newRatioFac, _ := newRatio.Mul(ratioFrom).Quo(ratioTo).ApproxSqrt()

	newRatioFrom := ratioFrom.Quo(newRatioFac)
	newRatioTo := ratioTo.Mul(newRatioFac)

	k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{Denom: ctx.TradeDenomGiving, Ratio: newRatioFrom})
	k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{Denom: ctx.TradeDenomReceiving, Ratio: newRatioTo})
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

func GetTradeLiquidities(denomGiving, denomReceiving string, cutLiquidity types.CutLiquidity) (math.LegacyDec, math.LegacyDec) {
	return cutLiquidity.GetFull(denomGiving), cutLiquidity.GetFull(denomReceiving)
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
	liqFrom := ctx.CutLiquidity.GetFull(ctx.TradeDenomGiving)
	liqTo := ctx.CutLiquidity.GetFull(ctx.TradeDenomReceiving)

	maxPrice := ctx.MaxPrice.Mul(math.LegacyOneDec().Sub(ctx.Fee))
	return ctx.CalcTradableAmountGivenPrice(liqFrom, liqTo, maxPrice)
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
	var max1, max2 *math.LegacyDec
	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		max2 = k.CalculateSingleSellableAmount(ctx.CutLiquidity, constants.BaseCurrency, ctx.TradeDenomReceiving, nil)
	}

	if max2 != nil && max2.IsZero() {
		zeroInt := math.ZeroInt()
		return &zeroInt, nil
	}

	if ctx.TradeDenomGiving != constants.BaseCurrency {
		max1 = k.CalculateSingleSellableAmount(ctx.CutLiquidity, ctx.TradeDenomGiving, constants.BaseCurrency, max2)
	} else {
		max1 = max2
	}

	if max1 == nil {
		return nil, nil
	}

	maximum := max1.TruncateInt()
	return &maximum, nil
}

// CalculateSingleSellableAmount calculates the maximum trading amount for a given trading pair, i.e. how much of
// denomFrom can be given at maximum. When there is no virtual liquidity, the tradable amount is infinity, thus the
// return amount is nil.
func (k Keeper) CalculateSingleSellableAmount(cutLiquidity types.CutLiquidity, denomFrom, denomTo string, maximumActual *math.LegacyDec) *math.LegacyDec {
	actualFrom := cutLiquidity.Get(denomFrom)
	actualTo := cutLiquidity.Get(denomTo)
	virtualFrom := cutLiquidity.GetVirtual(denomFrom)
	virtualTo := cutLiquidity.GetVirtual(denomTo)

	if maximumActual != nil && maximumActual.LT(actualTo) {
		virtualTo = actualTo.Add(virtualTo).Sub(*maximumActual)
		actualTo = *maximumActual
	}

	if virtualTo.IsNil() || !virtualTo.IsPositive() {
		return nil
	}

	return CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, maximumActual)
}

func CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec, maximumActual *math.LegacyDec) *math.LegacyDec {
	if virtualTo.IsNil() || virtualTo.IsZero() {
		return nil
	}

	X := actualFrom.Add(virtualFrom)
	maximum := X.Mul(actualTo.Quo(virtualTo)) // C
	return &maximum
}

// CalculateMaximumBuyableAmount...
func (k Keeper) CalculateMaximumBuyableAmount(ctx types.TradeContext) (*math.Int, error) {
	orderFee := ctx.OrdersCaches.OrderFee.Get()

	if ctx.HasOneStep() {
		maximum := k.CalculateSingleBuyableAmount(ctx.OrdersCaches, ctx.CutLiquidity, ctx.TradeDenomReceiving)
		maximum = subtractOrderFee(maximum, orderFee, ctx.IsOrder)
		return &maximum, nil
	}

	maxBase := k.CalculateSingleBuyableAmount(ctx.OrdersCaches, ctx.CutLiquidity, constants.BaseCurrency)
	maxBase = subtractOrderFee(maxBase, orderFee, ctx.IsOrder)

	maximum, _, err := calculateSingleTrade(constants.BaseCurrency, ctx.TradeDenomReceiving, maxBase.ToLegacyDec(), ctx.StepFee(), ctx.CutLiquidity, constant_product.ConstantProductTradeSell)
	if err != nil {
		return nil, err
	}

	maximumInt := maximum.TruncateInt()
	available := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving)
	maximumInt = math.MinInt(maximumInt, available)

	return &maximumInt, nil
}

func subtractOrderFee(amount math.Int, orderFee math.LegacyDec, isOrder bool) math.Int {
	if !isOrder {
		return amount
	}

	feeAmount := amount.ToLegacyDec().Quo(math.LegacyOneDec().Sub(orderFee)).Sub(amount.ToLegacyDec()).TruncateInt() // C
	return amount.Sub(feeAmount)
}

func (k Keeper) CalculateSingleBuyableAmount(ordersCaches *types.OrdersCaches, cutLiquidity types.CutLiquidity, denomReceiving string) math.Int {
	actualTo := ordersCaches.LiquidityPool.Get().AmountOf(denomReceiving).ToLegacyDec()
	virtualTo := cutLiquidity.GetVirtual(denomReceiving)

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
		return types.TradeSimulationResult{}, types.ErrDenomNotFound
	}

	ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	if err := k.PrepareCutLiquidity(&ctx); err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("preparing cutliquidity: %v", err)
	}

	var (
		amountIntermediate math.LegacyDec
		result             math.LegacyDec
		err                error
	)

	if ctx.DoFirstStep() {
		amountIntermediate, _, err = calculateSingleTrade(ctx.FirstGiving(), ctx.FirstReceiving(), ctx.TradeAmount.ToLegacyDec(), ctx.CalcStepFee(fee), ctx.CutLiquidity, cpTrade)
		if err != nil {
			return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
		}

		if ctx.HasTwoSteps() {
			amountIntermediateInt := amountIntermediate.TruncateInt()
			if ctx.TradeType == types.TradeTypeSell {
				ctx.CutLiquidity.UpdateBase(ctx.TradeType, ctx.TradeAmount, amountIntermediateInt)
				defer ctx.CutLiquidity.UpdateBase(ctx.TradeType, ctx.TradeAmount, amountIntermediateInt.Neg())
			} else {
				ctx.CutLiquidity.UpdateBase(ctx.TradeType, amountIntermediateInt, ctx.TradeAmount)
				defer ctx.CutLiquidity.UpdateBase(ctx.TradeType, amountIntermediateInt.Neg(), ctx.TradeAmount)
			}
		}
	} else {
		amountIntermediate = ctx.TradeAmount.ToLegacyDec()
	}

	if ctx.DoSecondStep() {
		result, _, err = calculateSingleTrade(ctx.SecondGiving(), ctx.SecondReceiving(), amountIntermediate, ctx.CalcStepFee(fee), ctx.CutLiquidity, cpTrade)
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
func CalculateSingleSell(denomGiving, denomReceiving string, offer, fee math.LegacyDec, cutLiquidity types.CutLiquidity) (math.LegacyDec, math.LegacyDec, error) {
	return calculateSingleTrade(denomGiving, denomReceiving, offer, fee, cutLiquidity, constant_product.ConstantProductTradeSell)
}

func CalculateSingleBuy(denomGiving, denomReceiving string, requested, fee math.LegacyDec, cutLiquidity types.CutLiquidity) (math.LegacyDec, math.LegacyDec, error) {
	return calculateSingleTrade(denomGiving, denomReceiving, requested, fee, cutLiquidity, constant_product.ConstantProductTradeBuy)
}

func calculateSingleTrade(denomGiving, denomReceiving string, offer, fee math.LegacyDec, cutLiquidity types.CutLiquidity, cpTrade constant_product.ConstantProductTrade) (math.LegacyDec, math.LegacyDec, error) {
	if denomGiving == denomReceiving {
		return offer, math.LegacyZeroDec(), nil
	}

	poolFrom, poolTo := GetTradeLiquidities(denomGiving, denomReceiving, cutLiquidity)
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
		return types.ErrDenomNotFound
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomReceiving) {
		return types.ErrDenomNotFound
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
