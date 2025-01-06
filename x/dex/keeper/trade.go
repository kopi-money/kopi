package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	"github.com/kopi-money/kopi/x/dex/types"
)

type notEnoughBuyFundsError struct {
	amount math.Int
}

func (e notEnoughBuyFundsError) Error() string {
	return ""
}

func (k Keeper) ExecuteSell(ctx types.TradeContext) (types.TradeResult, error) {
	ctx.TradeType = types.TradeTypeSell
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumSellableAmount
	ctx.CalcTradableAmountGivenPrice = constant_product.CalculateMaximumGiving
	ctx.CalcAmountToGive = func() math.Int {
		return ctx.TradeAmount
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
	ctx.TradeType = types.TradeTypeBuy
	ctx.CalcMaximumTradableAmount = k.CalculateMaximumBuyableAmount
	ctx.CalcTradableAmountGivenPrice = constant_product.CalculateMaximumReceiving
	ctx.CalcAmountToGive = func() math.Int {
		tradeResult, _ := k.SimulateBuy(ctx)
		return tradeResult.AmountGiven
	}

	ctx.CalcMaximumTradeAmountByWallet = func() (math.Int, error) {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		available := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount

		poolFrom, poolTo := k.GetFullLiquidityBaseOtherCache(ctx.GetOrdersCaches(), ctx.TradeDenomGiving, ctx.TradeDenomReceiving)
		amountToReceive, _, err := constant_product.ConstantProductTradeSell(poolFrom, poolTo, available.ToLegacyDec(), ctx.FullFee())
		if err != nil {
			return math.Int{}, err
		}

		return amountToReceive.TruncateInt(), nil
	}

	ctx.IntermediateTradeAmount = types.IntermediateTradeAmountUsed

	if ctx.MaximumAvailableAmount.IsNil() {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		ctx.MaximumAvailableAmount = k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount
	}

	result, err := k.executeTrade(&ctx)
	if err != nil {
		var sell notEnoughBuyFundsError
		if errors.As(err, &sell) {
			return k.ExecuteSell(ctx.ToSell(sell.amount))
		}

		return types.TradeResult{}, err
	}

	return result.Get(types.TradeTypeBuy), nil
}

func (k Keeper) executeTrade(ctx *types.TradeContext) (types.TradeResults, error) {
	if err := k.validateTradeOptions(ctx); err != nil {
		return types.TradeResults{}, fmt.Errorf("error in trade options: %w", err)
	}

	if ctx.Fee.IsNil() {
		ctx.Fee = k.getTradeFee(ctx, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
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

	// When selling:
	// With the given funds and the liquidity on the DEX, we can calculate how much a user is to receive when trading.
	// In some cases though, caused by virtual liquidity, the user would receive more than there is liquidity present.
	// In those cases, the given amount is lowered if the user is okay with an incomplete trade. If not, an error is
	// returned.
	// When buying:
	// Given how much funds are in the user's wallet, the user might not get the full desired amount.

	maximumTradableAmount := ctx.CalcMaximumTradableAmount(*ctx)
	if maximumTradableAmount != nil && maximumTradableAmount.LT(ctx.TradeAmount) {
		if ctx.MinimumTradeAmount != nil && maximumTradableAmount.LT(*ctx.MinimumTradeAmount) {
			if ctx.TradeType == types.TradeTypeSell {
				return types.TradeResults{}, types.ErrNotEnoughLiquidity
			} else {
				return types.TradeResults{}, types.ErrNotEnoughFunds
			}
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
		priceAmount := k.calculateAmountGivenPrice(ctx.OrdersCaches, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, *ctx.MaxPrice, ctx.Fee, ctx.CalcTradableAmountGivenPrice).TruncateInt()
		if !priceAmount.IsPositive() {
			return types.TradeResults{}, types.ErrNegativeTradeAmount
		}

		if priceAmount.LT(ctx.TradeAmount) {
			if ctx.MinimumTradeAmount.GT(priceAmount) {
				return types.TradeResults{}, types.ErrPriceTooLow
			}

			ctx.TradeAmount = priceAmount
		}
	}

	// If the user does not have enough funds given the trade amount, an error is returned. We skip that check for
	// orders because we can assume that the order pool balance is correct.
	if !ctx.IsOrder {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)

		available := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount
		if available.LT(ctx.CalcAmountToGive()) {
			// If the desired amount to buy would require to spend more than what's in the address's wallet, the trade
			// is turned into a sell using all the available funds.
			if ctx.TradeType == types.TradeTypeBuy {
				return types.TradeResults{}, notEnoughBuyFundsError{amount: available}
			}

			var err error
			ctx.TradeAmount, err = ctx.CalcMaximumTradeAmountByWallet()
			if err != nil {
				return types.TradeResults{}, fmt.Errorf("calculating maximum trade amount: %w", err)
			}

			if ctx.MinimumTradeAmount != nil && ctx.TradeAmount.LT(*ctx.MinimumTradeAmount) {
				return types.TradeResults{}, types.ErrPriceTooLow
			}
		}
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

	return types.TradeResults{
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
	}, nil
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
	fullBase, fullOther, fullFrom, fullTo := k.GetLiquidities(ctx)
	amountToGiveGross, feeGiving, amountToReceiveGross, feeReceiving, err := k.calculateTradeAmounts(ctx, fullFrom, fullTo, ctx.TradeAmount.ToLegacyDec(), ctx.StepFee())
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not calculate trade amounts: %w", err)
	}

	liquidityProviders, amountToReceiveLeft, err := k.determineLiquidityProviders(ctx, amountToReceiveGross.Add(feeReceiving), ctx.StepDenomReceiving)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not send from source to dex (2): %w", err)
	}
	amountActuallyReceivedGross := amountToReceiveGross.Sub(amountToReceiveLeft)

	shareUsed := math.LegacyZeroDec()
	if amountActuallyReceivedGross.GT(math.ZeroInt()) {
		shareUsed = amountActuallyReceivedGross.ToLegacyDec().Quo(amountToReceiveGross.ToLegacyDec())
	}

	amountUsedNet := shareUsed.Mul(amountToGiveGross.ToLegacyDec()).RoundInt()

	if ctx.StepDenomGiving != constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, ctx.StepDenomGiving, amountUsedNet)
	}

	var feePaid, feeForReserve, feeForLiquidityProviders math.Int
	var feeDenom string

	if ctx.TradeType == types.TradeTypeSell {
		feePaid, feeForReserve, feeForLiquidityProviders = manageFee(feeReceiving, ctx.ReserveFeeShare)
		feeDenom = ctx.StepDenomReceiving
	} else {
		feePaid, feeForReserve, feeForLiquidityProviders = manageFee(feeGiving, ctx.ReserveFeeShare)
		feeDenom = ctx.StepDenomGiving
	}

	if err = k.distributeFeeForLiquidityProviders(ctx, liquidityProviders, feeForLiquidityProviders, feeDenom); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not distribute TO funds to liquidity providers: %w", err)
	}

	if err = k.distributeGivenFunds(ctx, ctx.OrdersCaches, liquidityProviders, amountToGiveGross.Sub(feeGiving), ctx.StepDenomGiving); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, fmt.Errorf("could not distribute FROM funds to liquidity providers: %w", err)
	}

	ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolReserve, feeDenom, feeForReserve)
	payoutAmount := ctx.TradeBalances.NetBalance(accPoolTrade, ctx.StepDenomReceiving)
	if ctx.StepDenomReceiving != constants.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomReceiving, payoutAmount)
	}

	poolFrom2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomGiving)
	poolTo2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomReceiving)
	changeFrom := poolFrom2.Sub(poolFrom1)
	changeTo := poolTo2.Sub(poolTo1)

	if ctx.StepDenomGiving != constants.BaseCurrency {
		k.updateRatio(ctx.TradeContext.Context, ctx.StepDenomGiving, fullBase, fullOther, changeTo, changeFrom)
	}

	if ctx.StepDenomReceiving != constants.BaseCurrency {
		k.updateRatio(ctx.TradeContext.Context, ctx.StepDenomReceiving, fullBase, fullOther, changeFrom, changeTo)
	}

	return amountToGiveGross, payoutAmount, feePaid, nil
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

func (k Keeper) GetLiquidities(ctx types.TradeStepContext) (math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	return k.getLiquidities(ctx.OrdersCaches, ctx.StepDenomGiving, ctx.StepDenomReceiving)
}

func (k Keeper) getLiquidities(ordersCaches *types.OrdersCaches, denomGiving, denomReceiving string) (math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	var otherDenom string
	if denomGiving == constants.BaseCurrency {
		otherDenom = denomReceiving
	} else {
		otherDenom = denomGiving
	}

	fullBase := k.GetFullLiquidityBaseCache(ordersCaches, otherDenom)
	fullOther := k.GetFullLiquidityOtherCache(ordersCaches, otherDenom)

	var fullFrom, fullTo math.LegacyDec
	if denomGiving == constants.BaseCurrency {
		fullFrom, fullTo = fullBase, fullOther
	} else {
		fullFrom, fullTo = fullOther, fullBase
	}

	return fullBase, fullOther, fullFrom, fullTo
}

func (k Keeper) handleOrderFee(ordersCaches *types.OrdersCaches, tradeBalances types.TradeBalances, orderFee math.LegacyDec, amount math.Int, isBuy bool) math.Int {
	var feeAmount math.Int

	if isBuy {
		feeAmount = amount.ToLegacyDec().Quo(math.LegacyOneDec().Sub(orderFee)).Sub(amount.ToLegacyDec()).TruncateInt()
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

func (k Keeper) calculateAmountGivenPrice(ordersCaches *types.OrdersCaches, denomFrom, denomTo string, maxPrice, fee math.LegacyDec, calc constant_product.CalculateMaximumAmount) math.LegacyDec {
	liqFrom := k.GetFullLiquidity(ordersCaches, denomFrom, denomTo)
	liqTo := k.GetFullLiquidity(ordersCaches, denomTo, denomFrom)
	return calc(liqFrom, liqTo, maxPrice, fee)
}

func (k Keeper) getSpendableCoins(ctx context.Context, address sdk.AccAddress, denom string) math.Int {
	for _, coins := range k.BankKeeper.SpendableCoins(ctx, address) {
		if coins.Denom == denom {
			return coins.Amount
		}
	}

	return math.ZeroInt()
}

// updateRatio sets the ratio between a given denom and the base currency
func (k Keeper) updateRatio(ctx context.Context, denom string, fullBase, fullOther math.LegacyDec, changeBase, changeOther math.Int) {
	if denom == constants.BaseCurrency {
		panic("only to be called for non-base denoms")
	}

	fullBase = fullBase.Add(changeBase.ToLegacyDec())
	fullOther = fullOther.Add(changeOther.ToLegacyDec())

	if fullBase.IsPositive() {
		k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{
			Denom: denom,
			Ratio: fullOther.Quo(fullBase),
		})
	}
}

// SimulateTradeForReserve is used when calculating the profitability of a mint/burn trade. When trading, the reserve
// has to pay the trade fee. However, part of it will be paid out to itself. Thus, when estimating the profitability of
// a trade, that part of the fee is removed.
func (k Keeper) SimulateTradeForReserve(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	reserveShare := k.GetParams(ctx).ReserveShare
	fee := k.getTradeFee(ctx, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	fee = fee.Mul(math.LegacyOneDec().Sub(reserveShare))
	return k.SimulateSellWithFee(ctx, fee)
}

// CalculateMaximumSellableAmount calculates the maximum sellable amount for a given trading pair while routing the
// trade via the base currency. First, the tradable amount between the base currency and the "to" currency is
// calculated. In the second step, the tradable amount from the "from" currency to the base currency is calculated. The
// previously calculated maximum tradable amount is given to that function to cover cases where the size bottleneck is
// in the second trading step.
func (k Keeper) CalculateMaximumSellableAmount(ctx types.TradeContext) *math.Int {
	var max1, max2 *math.LegacyDec
	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		max2 = k.CalculateSingleSellableAmount(ctx.OrdersCaches, constants.BaseCurrency, ctx.TradeDenomReceiving, nil)
	}

	if ctx.TradeDenomGiving != constants.BaseCurrency {
		max1 = k.CalculateSingleSellableAmount(ctx.OrdersCaches, ctx.TradeDenomGiving, constants.BaseCurrency, max2)
	} else {
		max1 = max2
	}

	if max1 == nil {
		return nil
	}

	maximum := max1.TruncateInt()
	return &maximum
}

// CalculateSingleSellableAmount calculates the maximum trading amount for a given trading pair, i.e. how much of
// denomFrom can be given at maximum. When there is no virtual liquidity, the tradable amount is infinity, thus the
// return amount is nil.
func (k Keeper) CalculateSingleSellableAmount(ordersCaches *types.OrdersCaches, denomGiving, denomReceiving string, maximumActual *math.LegacyDec) *math.LegacyDec {
	actualFrom := ordersCaches.LiquidityPool.Get().AmountOf(denomGiving).ToLegacyDec()
	actualTo := ordersCaches.LiquidityPool.Get().AmountOf(denomReceiving).ToLegacyDec()

	var virtualFrom, virtualTo math.LegacyDec
	if denomGiving == constants.BaseCurrency {
		pair := ordersCaches.LiquidityPair.Get(denomReceiving)
		virtualFrom = pair.VirtualBase
		virtualTo = pair.VirtualOther
	} else {
		pair := ordersCaches.LiquidityPair.Get(denomGiving)
		virtualTo = pair.VirtualBase
		virtualFrom = pair.VirtualOther
	}

	return CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, maximumActual)
}

func CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec, maximumActual *math.LegacyDec) *math.LegacyDec {
	if maximumActual != nil && maximumActual.LT(actualTo) {
		virtualTo = actualTo.Add(virtualTo).Sub(*maximumActual)
		actualTo = *maximumActual
	}

	if virtualTo.IsNil() || virtualTo.IsZero() {
		return nil
	}

	maximumTradable := calculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	if maximumTradable == nil {
		return nil
	}

	return maximumTradable
}

func calculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec) *math.LegacyDec {
	if virtualTo.IsZero() {
		return nil
	}

	X := actualFrom.Add(virtualFrom)
	maximum := X.Mul(actualTo.Quo(virtualTo))
	return &maximum
}

// CalculateMaximumBuyableAmount...
func (k Keeper) CalculateMaximumBuyableAmount(ctx types.TradeContext) *math.Int {
	var maximum math.LegacyDec
	if ctx.TradeDenomGiving != constants.BaseCurrency {
		maximum = ctx.OrdersCaches.LiquidityPool.Get().AmountOf(constants.BaseCurrency).ToLegacyDec()
		if ctx.IsOrder {
			maximum = maximum.Mul(math.LegacyOneDec().Sub(ctx.OrdersCaches.OrderFee.Get()))
		}
	}

	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		poolSize := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving).ToLegacyDec()
		if maximum.IsNil() {
			maximum = poolSize
		} else {
			amountReceived, _, _ := k.CalculateSingleSell(ctx, constants.BaseCurrency, ctx.TradeDenomReceiving, maximum, math.LegacyZeroDec())
			maximum = math.LegacyMinDec(poolSize, amountReceived)
		}
	}

	maximumInt := maximum.TruncateInt()
	return &maximumInt
}

func (k Keeper) CalculateSingleBuyableAmount(ordersCaches *types.OrdersCaches, denomGiving, denomReceiving string, maximumActual *math.LegacyDec) *math.LegacyDec {
	actualFrom := ordersCaches.LiquidityPool.Get().AmountOf(denomGiving).ToLegacyDec()
	actualTo := ordersCaches.LiquidityPool.Get().AmountOf(denomReceiving).ToLegacyDec()

	var virtualFrom, virtualTo math.LegacyDec
	if denomGiving == constants.BaseCurrency {
		pair := ordersCaches.LiquidityPair.Get(denomReceiving)
		virtualFrom = pair.VirtualBase
		virtualTo = pair.VirtualOther
	} else {
		pair := ordersCaches.LiquidityPair.Get(denomGiving)
		virtualTo = pair.VirtualBase
		virtualFrom = pair.VirtualOther
	}

	return CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, maximumActual)
}

func (k Keeper) checkTradePoolLiquidities(ordersCaches *types.OrdersCaches, denomFrom, denomTo string) error {
	if denomFrom != constants.BaseCurrency {
		if err := k.checkPoolLiquidities(ordersCaches, denomFrom, constants.BaseCurrency); err != nil {
			return err
		}
	}

	if denomTo != constants.BaseCurrency {
		if err := k.checkPoolLiquidities(ordersCaches, constants.BaseCurrency, denomTo); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) checkPoolLiquidities(ordersCaches *types.OrdersCaches, denomFrom, denomTo string) error {
	poolFrom, poolTo := k.GetFullLiquidityBaseOtherCache(ordersCaches, denomFrom, denomTo)

	if poolTo.IsZero() {
		return types.ErrNotEnoughLiquidity
	}
	if poolFrom.IsZero() {
		return types.ErrNotEnoughLiquidity
	}

	return nil
}

func (k Keeper) addProviderFee(ctx context.Context, amount, tradeFee math.LegacyDec) math.LegacyDec {
	feeShareReserve := k.GetReserveFeeShare(ctx)
	feeShareProvider := math.LegacyOneDec().Sub(feeShareReserve)
	feeProvider := tradeFee.Mul(feeShareProvider)

	return amount.Quo(math.LegacyOneDec().Sub(feeProvider))
}

// manageFee is called each time liquidity is used for a trade. amount indicates how much is traded right now, the
// address corresponds to the user whose liquidity is used right now.
func manageFee(feeAmount math.Int, reserveFeeShare math.LegacyDec) (math.Int, math.Int, math.Int) {
	feeForReserve := feeAmount.ToLegacyDec().Mul(reserveFeeShare).RoundInt()
	feeForLiquidityProviders := feeAmount.Sub(feeForReserve)
	return feeAmount, feeForReserve, feeForLiquidityProviders
}

func (k Keeper) SimulateSell(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	fee := k.getTradeFee(ctx, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	return k.SimulateSellWithFee(ctx, fee)
}

func (k Keeper) SimulateSellWithFee(ctx types.TradeContext, fee math.LegacyDec) (types.TradeSimulationResult, error) {
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

	amount := ctx.TradeAmount.ToLegacyDec()
	amountFee := amount.Mul(fee)
	amount = amount.Sub(amountFee)

	amountIntermediate, _, err := k.CalculateSingleSell(ctx, ctx.TradeDenomGiving, constants.BaseCurrency, amount, math.LegacyZeroDec())
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
	}

	amountReceived, _, err := k.CalculateSingleSell(ctx, constants.BaseCurrency, ctx.TradeDenomReceiving, amountIntermediate, math.LegacyZeroDec())
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
	}

	return types.TradeSimulationResult{
		AmountIntermediate: amountIntermediate.TruncateInt(),
		AmountGiven:        ctx.TradeAmount,
		AmountReceived:     amountReceived.TruncateInt(),
		FeeGiven:           amountFee.TruncateInt(),
	}, nil
}

// CalculateSingleSell is used when simulating a sell. Since the sell is not executed, i.e. no liquidity is changed,
// this method does not need to iterate over the liquidity list but can simply calculate everything.
func (k Keeper) CalculateSingleSell(ctx context.Context, denomGiving, denomReceiving string, offer, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	return k.calculateSingleTrade(ctx, denomGiving, denomReceiving, offer, fee, constant_product.ConstantProductTradeSell)
}

func (k Keeper) CalculateSingleBuy(ctx context.Context, denomGiving, denomReceiving string, requested, fee math.LegacyDec) (math.LegacyDec, math.LegacyDec, error) {
	return k.calculateSingleTrade(ctx, denomGiving, denomReceiving, requested, fee, constant_product.ConstantProductTradeBuy)
}

func (k Keeper) calculateSingleTrade(ctx context.Context, denomGiving, denomReceiving string, offer, fee math.LegacyDec, cpTrade constant_product.ConstantProductTrade) (math.LegacyDec, math.LegacyDec, error) {
	if denomGiving == denomReceiving {
		return offer, math.LegacyZeroDec(), nil
	}

	var poolFrom, poolTo math.LegacyDec

	if denomGiving == constants.BaseCurrency {
		poolFrom = k.GetFullLiquidityBase(ctx, denomReceiving)
		poolTo = k.GetFullLiquidityOther(ctx, denomReceiving)
	} else {
		poolFrom = k.GetFullLiquidityOther(ctx, denomGiving)
		poolTo = k.GetFullLiquidityBase(ctx, denomGiving)
	}

	if poolFrom.IsZero() {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("no liquidity for: %v", denomGiving)
	}

	if poolTo.IsZero() {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("no liquidity for: %v", denomReceiving)
	}

	amount, feeAmount, err := cpTrade(poolFrom, poolTo, offer, fee)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, err
	}

	return amount, feeAmount, nil
}

func (k Keeper) SimulateBuy(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	fee := k.getTradeFee(ctx, ctx.DiscountAddress, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, ctx.ExcludeFromDiscount)
	return k.SimulateBuyWithFee(ctx, fee)
}

func (k Keeper) SimulateBuyWithFee(ctx types.TradeContext, fee math.LegacyDec) (types.TradeSimulationResult, error) {
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

	amountToIntermediate, _, err := k.CalculateSingleBuy(ctx, constants.BaseCurrency, ctx.TradeDenomReceiving, ctx.TradeAmount.ToLegacyDec(), math.LegacyZeroDec())
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
	}

	amountToGive, _, err := k.CalculateSingleBuy(ctx, ctx.TradeDenomGiving, constants.BaseCurrency, amountToIntermediate, math.LegacyZeroDec())
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("could not calculate single trade #1: %w", err)
	}

	amountFee := amountToGive.Mul(fee)
	amountToGive = amountToGive.Add(amountFee)

	return types.TradeSimulationResult{
		AmountIntermediate: amountToIntermediate.TruncateInt(),
		AmountGiven:        amountToGive.TruncateInt(),
		AmountReceived:     ctx.TradeAmount,
		FeeGiven:           amountFee.TruncateInt(),
	}, nil
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
