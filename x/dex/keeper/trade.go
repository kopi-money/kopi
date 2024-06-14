package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

// ExecuteTrade is called when a user sends a tx to execute and sets incomplete=true. First, a trade to the base
// currency is executed, then a trade from the base currency to the target currency.
func (k Keeper) ExecuteTrade(ctx types.TradeContext) (math.Int, math.Int, math.Int, math.Int, math.Int, error) {
	hasBalances := ctx.TradeBalances != nil
	if !hasBalances {
		tradeBalances := NewTradeBalances()
		ctx.TradeBalances = tradeBalances
	}

	i1, i2, i3, i4, i5, err := k.executeTrade(ctx)
	if err != nil {
		return i1, i2, i3, i4, i5, err
	}

	if !hasBalances {
		if err = ctx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
			return i1, i2, i3, i4, i5, errors.Wrap(err, "could not settle balances")
		}
	}

	return i1, i2, i3, i4, i5, nil
}

func (k Keeper) executeTrade(ctx types.TradeContext) (math.Int, math.Int, math.Int, math.Int, math.Int, error) {
	if err := k.validateTradeOptions(&ctx); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "error in trade options")
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

	tradeFee := k.getTradeFee(ctx)

	// With the given funds and the liquidity on the DEX, we can calculate how much a user is to receive when trading.
	// In some cases though, caused by virtual liquidity, the user would receive more than there is liquidity present.
	// In those caes, the given amount is lowered if the user is okay with an incomplete trade. If not, an error is
	// returned.
	maximumTradableAmount, err := k.CalculateMaximumTradableAmount(ctx, ctx.OrdersCaches, tradeFee, ctx.TradeDenomStart, ctx.TradeDenomEnd)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not calculate maximum tradable amount")
	}

	if maximumTradableAmount.LT(ctx.GivenAmount) {
		if maximumTradableAmount.GT(math.ZeroInt()) && ctx.AllowIncomplete {
			ctx.GivenAmount = maximumTradableAmount
		} else {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughLiquidity
		}
	}

	// When a maximum price is set, it is checked how much has to be given to achieve the maximum price. If that amount
	// is lower than what it is intended to be given, it means trading with the intended amount would result in a higher
	// price than wanted. In that case, the trade amount is either lowered when the user accepts an incomplete trade, or
	// an error is returned.
	if ctx.MaxPrice != nil {
		priceAmount := k.calculateAmountGivenPrice(ctx.OrdersCaches, ctx.TradeDenomStart, ctx.TradeDenomEnd, *ctx.MaxPrice, tradeFee).TruncateInt()
		if priceAmount.LTE(math.ZeroInt()) {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrPriceTooLow
		}

		if priceAmount.LT(ctx.GivenAmount) {
			if ctx.AllowIncomplete {
				ctx.GivenAmount = priceAmount
			} else {
				return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrPriceTooLow
			}
		}
	}

	//if ctx.TradeDenomEnd == utils.BaseCurrency {
	//	poolFrom, poolTo = k.GetFullLiquidityBaseOther(ctx, ctx.TradeDenomStart, ctx.TradeDenomEnd)
	//	amountToReceive = ctx.TradeCalculation.Forward(poolFrom, poolTo, ctx.GivenAmount.ToLegacyDec())
	//	poolSize = ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomEnd)
	//	if !wasBefore && amountToReceive.GT(poolSize) {
	//		fmt.Println("!!!")
	//	}
	//}

	// If the trade amount is too small, an error is returned. The reason for that is that small trade amounts are more
	// affected by rounding issues.
	if ctx.GivenAmount.LT(math.NewInt(1000)) {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrTradeAmountTooSmall
	}

	// If the user doesn not have enough funds given the trade amount, an error is returned.
	if err := k.checkSpendableCoins(ctx, ctx.CoinSource, ctx.TradeDenomStart, ctx.GivenAmount); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughFunds
	}

	// Additional check whether there is enough liquidity
	if err := k.checkTradePoolLiquidities(ctx); err != nil {
		if ctx.AllowIncomplete {
			return math.Int{}, math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), nil
		} else {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughLiquidity
		}
	}

	// First trade step from the starting currency to the base currency
	amountUsed1, amountReceived1, feePaid1, err := k.ExecuteTradeStep(ctx.TradeToBase(tradeFee))
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), errors.Wrap(err, "could not execute trade step 1")
	}

	// Reimburse the user for the trade fee
	reimbursement := k.reimburseFee(ctx, ctx.OrdersCaches, ctx.TradeBalances, feePaid1)
	amountReceived1 = amountReceived1.Add(reimbursement)

	if ctx.IsOrder {
		amountReceived1, err = k.handleOrderFee(ctx.OrdersCaches, ctx.TradeBalances, ctx.OrdersCaches.OrderFee.Get(), amountReceived1)
		if err != nil {
			return math.Int{}, math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), errors.Wrap(err, "could not handle order fee")
		}
	}

	// Second trade from the base currency to the target currency
	_, amountReceived2, feePaid2, err := k.ExecuteTradeStep(ctx.TradeToTarget(tradeFee, amountReceived1))
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), errors.Wrap(err, "could not execute trade step 2")
	}

	ctx.OrdersCaches.Clear()
	k.AddTradeAmount(ctx, ctx.CoinTarget, amountReceived1)

	return amountUsed1, amountReceived1, amountReceived2, feePaid1, feePaid2, nil
}

func (k Keeper) handleOrderFee(ordersCaches *types.OrdersCaches, tradeBalances types.TradeBalances, orderFee math.LegacyDec, amount math.Int) (math.Int, error) {
	feeAmount := amount.ToLegacyDec().Mul(orderFee).RoundInt()
	tradeBalances.AddTransfer(
		ordersCaches.AccPoolTrade.Get().String(),
		ordersCaches.AccPoolReserve.Get().String(),
		utils.BaseCurrency, feeAmount,
	)

	return amount.Sub(feeAmount), nil
}

func (k Keeper) calculateAmountGivenPrice(ordersCaches *types.OrdersCaches, denomFrom, denomTo string, maxPrice, fee math.LegacyDec) math.LegacyDec {
	liqFrom := k.GetFullLiquidity(ordersCaches, denomFrom, denomTo)
	liqTo := k.GetFullLiquidity(ordersCaches, denomTo, denomFrom)
	maxPrice = maxPrice.Mul(math.LegacyOneDec().Sub(fee))
	return maxPrice.Mul(liqTo).Sub(liqFrom)
}

func (k Keeper) reimburseFee(ctx context.Context, ordersCaches *types.OrdersCaches, tradeBalances types.TradeBalances, feePaid math.Int) math.Int {
	if feePaid.Equal(math.ZeroInt()) {
		return math.ZeroInt()
	}

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFees)
	coinAvailable := k.getSpendableCoins(ctx, addr.GetAddress(), utils.BaseCurrency)

	reimburse := feePaid.ToLegacyDec().Mul(k.GetParams(ctx).FeeReimbursement).RoundInt()
	reimburse = math.MinInt(reimburse, coinAvailable)
	if reimburse.LTE(math.ZeroInt()) {
		return math.ZeroInt()
	}

	tradeBalances.AddTransfer(
		ordersCaches.AccPoolFees.Get().String(),
		ordersCaches.AccPoolTrade.Get().String(),
		utils.BaseCurrency, feePaid,
	)

	return reimburse
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
	if denom == utils.BaseCurrency {
		panic("only to be called for non-base denoms")
	}

	fullBase = fullBase.Add(changeBase.ToLegacyDec())
	fullOther = fullOther.Add(changeOther.ToLegacyDec())

	if fullBase.GT(math.LegacyZeroDec()) {
		k.SetRatio(ctx, types.Ratio{
			Denom: denom,
			Ratio: fullOther.Quo(fullBase),
		})
	}
}

func (k Keeper) TradeSimulation(ctx types.TradeContext) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	fee := k.getTradeFee(ctx)
	return k.SimulateTradeWithFee(ctx, fee)
}

// SimulateTradeForReserve is used when calculating the profitability of a mint/burn trade. When trading, the reserve
// has to pay the trade fee. However, part of it will be paid out to itself. Thus, when estimating the profitability of
// a trade, that part of the fee is removed.
func (k Keeper) SimulateTradeForReserve(ctx types.TradeContext) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	reserveShare := k.GetParams(ctx).ReserveShare
	fee := k.getTradeFee(ctx)
	fee = fee.Mul(math.LegacyOneDec().Sub(reserveShare))
	return k.SimulateTradeWithFee(ctx, fee)
}

func (k Keeper) SimulateTradeWithFee(ctx types.TradeContext, fee math.LegacyDec) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	if ctx.TradeDenomStart == ctx.TradeDenomEnd {
		return ctx.GivenAmount, math.LegacyZeroDec(), math.LegacyOneDec(), nil
	}

	if ctx.GivenAmount.Equal(math.ZeroInt()) {
		return math.ZeroInt(), math.LegacyZeroDec(), math.LegacyZeroDec(), nil
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomStart) || !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomEnd) {
		return math.Int{}, math.LegacyDec{}, math.LegacyDec{}, types.ErrDenomNotFound
	}

	amountStart := ctx.GivenAmount.ToLegacyDec()
	feePaid := amountStart.Mul(fee)
	amountStart = amountStart.Sub(feePaid)

	amountReceived, err := k.calculateSingleTrade(ctx, ctx.TradeDenomStart, utils.BaseCurrency, amountStart, fee)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, math.LegacyDec{}, errors.Wrap(err, "could not calculate single trade #1")
	}

	amountReceived, err = k.calculateSingleTrade(ctx, utils.BaseCurrency, ctx.TradeDenomEnd, amountReceived, fee)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, math.LegacyDec{}, errors.Wrap(err, "could not calculate single trade #1")
	}

	price := amountStart.Quo(amountReceived)
	return amountReceived.TruncateInt(), feePaid, price, nil
}

// CalculateMaximumTradableAmount calculates the maximum tradable amount for a given trading pair while routing the
// trade via the base currency. First, the tradable amount between the base currency and the "to" currency is
// calculated. In the second step, the tradable amount from the "from" currency to the base currency is calculated. The
// previously calculated maximum tradable amount is given to that function to cover cases where the size bottleneck is
// in the second trading step.
func (k Keeper) CalculateMaximumTradableAmount(ctx context.Context, ordersCaches *types.OrdersCaches, fee math.LegacyDec, denomFrom, denomTo string) (math.Int, error) {
	poolAmount := ordersCaches.LiquidityPool.Get().AmountOf(denomTo)

	tradeCtx := types.TradeContext{
		Context:         ctx,
		GivenAmount:     poolAmount,
		TradeDenomStart: denomTo,
		TradeDenomEnd:   denomFrom,
		OrdersCaches:    ordersCaches,
	}

	amount, _, _, err := k.SimulateTradeWithFee(tradeCtx, fee)
	if err != nil {
		return math.Int{}, errors.Wrap(err, "could not simulate trade")
	}

	return amount, nil

	//var max1, max2 *math.LegacyDec
	//if denomTo != utils.BaseCurrency {
	//	max2 = k.CalculateSingleMaximumTradableAmount(ordersCaches, utils.BaseCurrency, denomTo, nil)
	//}
	//
	//if denomFrom != utils.BaseCurrency {
	//	max1 = k.CalculateSingleMaximumTradableAmount(ordersCaches, denomFrom, utils.BaseCurrency, max2)
	//} else {
	//	max1 = max2
	//}
	//
	//if max1 == nil {
	//	return nil
	//}
	//
	//maximum := max1.TruncateInt()
	//return &maximum
}

// CalculateSingleMaximumTradableAmount calculates the maximum trading amount for a given trading pair, i.e. how much of
// denomFrom can be given at maximum. When there is no virtual liquidity, the tradable amount is infinity, thus the
// return amount is nil.
func (k Keeper) CalculateSingleMaximumTradableAmount(ordersCaches *types.OrdersCaches, denomFrom, denomTo string, maximumActual *math.LegacyDec) *math.LegacyDec {
	actualFrom := ordersCaches.LiquidityPool.Get().AmountOf(denomFrom).ToLegacyDec()
	actualTo := ordersCaches.LiquidityPool.Get().AmountOf(denomTo).ToLegacyDec()

	var virtualFrom, virtualTo math.LegacyDec
	if denomFrom == utils.BaseCurrency {
		pair := ordersCaches.LiquidityPair.Get(denomTo)
		virtualFrom = pair.VirtualBase
		virtualTo = pair.VirtualOther
	} else {
		pair := ordersCaches.LiquidityPair.Get(denomFrom)
		virtualTo = pair.VirtualBase
		virtualFrom = pair.VirtualOther
	}

	return CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo, maximumActual, &ordersCaches.MaximumTradableAmount)
}

func CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec, maximumActual *math.LegacyDec, cache *map[string]*math.LegacyDec) *math.LegacyDec {
	if maximumActual != nil && maximumActual.LT(actualTo) {
		virtualTo = actualTo.Add(virtualTo).Sub(*maximumActual)
		actualTo = *maximumActual
	}

	if virtualTo.Equal(math.LegacyZeroDec()) {
		return nil
	}

	var maximumTradable *math.LegacyDec
	if cache != nil {
		// Believe or not, but creating a key this way and checking whether this exact calculation has been done before
		// is faster than doing the actual calculation a second time...
		key := fmt.Sprintf("%v:%v:%v:%v", actualFrom.String(), actualTo.String(), virtualFrom.String(), virtualTo.String())
		maximum, has := (*cache)[key]
		if !has {
			maximum = calculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
			(*cache)[key] = maximum
		}

		maximumTradable = maximum
	} else {
		maximumTradable = calculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	}

	if maximumTradable == nil {
		return nil
	}

	return maximumTradable
}

func calculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec) *math.LegacyDec {
	if virtualTo.Equal(math.LegacyZeroDec()) {
		return nil
	}

	X := actualFrom.Add(virtualFrom)
	maximum := X.Mul(actualTo.Quo(virtualTo))
	return &maximum
}

func (k Keeper) checkTradePoolLiquidities(ctx types.TradeContext) error {
	if ctx.TradeDenomStart != utils.BaseCurrency {
		if err := k.checkPoolLiquidities(ctx, ctx.TradeDenomStart, utils.BaseCurrency); err != nil {
			return err
		}
	}

	if ctx.TradeDenomEnd != utils.BaseCurrency {
		if err := k.checkPoolLiquidities(ctx, utils.BaseCurrency, ctx.TradeDenomEnd); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) checkPoolLiquidities(ctx context.Context, denomFrom, denomTo string) error {
	poolFrom, poolTo := k.GetFullLiquidityBaseOther(ctx, denomFrom, denomTo)

	if poolTo.Equal(math.LegacyZeroDec()) {
		return types.ErrNotEnoughLiquidity
	}
	if poolFrom.Equal(math.LegacyZeroDec()) {
		return types.ErrNotEnoughLiquidity
	}

	return nil
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
	if ctx.StepDenomTo == utils.BaseCurrency && ctx.TradeDenomStart == utils.BaseCurrency {
		ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, utils.BaseCurrency, ctx.Amount)
		return ctx.Amount, ctx.Amount, math.ZeroInt(), nil
	}

	// If a trade is from something else to XKP, the following step sends XKP to the user in trade step 2
	if ctx.StepDenomFrom == utils.BaseCurrency && ctx.TradeDenomEnd == utils.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomTo, ctx.Amount)
		return ctx.Amount, ctx.Amount, math.ZeroInt(), nil
	}

	poolFrom1 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomFrom)
	poolTo1 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomTo)

	var otherDenom string
	if ctx.StepDenomFrom == utils.BaseCurrency {
		otherDenom = ctx.StepDenomTo
	} else {
		otherDenom = ctx.StepDenomFrom
	}

	fullBase := k.GetFullLiquidityBase(ctx, otherDenom)
	fullOther := k.GetFullLiquidityOther(ctx, otherDenom)

	// calculate how much the trader can receive with this liquidity entry
	poolFrom, poolTo := k.GetFullLiquidityBaseOther(ctx, ctx.StepDenomFrom, ctx.StepDenomTo)
	amountToReceive := ctx.TradeCalculation.Forward(poolFrom, poolTo, ctx.Amount.ToLegacyDec())
	if amountToReceive.Equal(math.ZeroInt()) {
		return math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), nil
	}

	liquidityProviders, amountToReceiveLeft, err := k.determineLiquidityProviders(ctx, amountToReceive, ctx.StepDenomTo)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from source to dex (2)")
	}
	amountReceivedGross := amountToReceive.Sub(amountToReceiveLeft)

	shareUsed := amountReceivedGross.ToLegacyDec().Quo(amountToReceive.ToLegacyDec())
	amountUsed := shareUsed.Mul(ctx.Amount.ToLegacyDec()).RoundInt()

	if amountUsed.GT(math.ZeroInt()) && ctx.StepDenomFrom != utils.BaseCurrency {
		ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, ctx.StepDenomFrom, amountUsed)
	}

	feePaid, feeForReserve, feeForLiquidityProviders := k.manageFee(ctx, amountReceivedGross, ctx.TradeFee)
	if err = k.distributeFeeForLiquidityProviders(ctx, liquidityProviders, feeForLiquidityProviders, ctx.StepDenomTo); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not distribute TO funds to liquidity providers")
	}

	if err = k.distributeGivenFunds(ctx, ctx.OrdersCaches, liquidityProviders, amountUsed, ctx.StepDenomFrom); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not distribute FROM funds to liquidity providers")
	}

	if feeForReserve.GT(math.ZeroInt()) {
		ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolReserve, ctx.StepDenomTo, feeForReserve)
	}

	payoutAmount := ctx.TradeBalances.NetBalance(ctx.OrdersCaches.AccPoolTrade.Get().String(), ctx.StepDenomTo)
	if payoutAmount.GT(math.ZeroInt()) && ctx.StepDenomTo != utils.BaseCurrency {
		ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.StepDenomTo, payoutAmount)
	}

	poolFrom2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomFrom)
	poolTo2 := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.StepDenomTo)
	changeFrom := poolFrom2.Sub(poolFrom1)
	changeTo := poolTo2.Sub(poolTo1)

	if ctx.StepDenomFrom != utils.BaseCurrency {
		k.updateRatio(ctx.TradeContext.Context, ctx.StepDenomFrom, fullBase, fullOther, changeTo, changeFrom)
	}

	if ctx.StepDenomTo != utils.BaseCurrency {
		k.updateRatio(ctx.TradeContext.Context, ctx.StepDenomTo, fullBase, fullOther, changeFrom, changeTo)
	}

	return amountUsed, payoutAmount, feePaid, nil
}

func (k Keeper) addProviderFee(ctx context.Context, amount, tradeFee math.LegacyDec) math.LegacyDec {
	feeShareReserve := k.GetReserveFeeShare(ctx)
	feeShareProvider := math.LegacyOneDec().Sub(feeShareReserve)
	feeProvider := tradeFee.Mul(feeShareProvider)

	return amount.Quo(math.LegacyOneDec().Sub(feeProvider))
}

// manageFee is called each time liquidity is used for a trade. amount indicates how much is traded right now, the
// address corresponds to the user whose liquidity is used right now.
func (k Keeper) manageFee(ctx context.Context, amount math.Int, tradeFee math.LegacyDec) (math.Int, math.Int, math.Int) {
	feeAmount := amount.ToLegacyDec().Mul(tradeFee)
	feeForReserve := feeAmount.Mul(k.GetReserveFeeShare(ctx))
	feeForLiquidityProviders := feeAmount.RoundInt().Sub(feeForReserve.RoundInt())
	return feeAmount.RoundInt(), feeForReserve.RoundInt(), feeForLiquidityProviders
}

// calculateSingleTrade is used when simulating a trade. Since the trade is no executed, i.e. no liquidity is changed,
// this method does not need to iterate over the liquidity list but can simply calculate everything.
func (k Keeper) calculateSingleTrade(ctx context.Context, denomFrom, denomTo string, offer, fee math.LegacyDec) (math.LegacyDec, error) {
	if denomFrom == denomTo {
		return offer, nil
	}

	var poolFrom, poolTo math.LegacyDec

	if denomFrom == utils.BaseCurrency {
		poolFrom = k.GetFullLiquidityBase(ctx, denomTo)
		poolTo = k.GetFullLiquidityOther(ctx, denomTo)
	} else {
		poolFrom = k.GetFullLiquidityOther(ctx, denomFrom)
		poolTo = k.GetFullLiquidityBase(ctx, denomFrom)
	}

	if poolFrom.IsZero() {
		return math.LegacyDec{}, fmt.Errorf("no liquidity for: %v", denomFrom)
	}

	if poolTo.IsZero() {
		return math.LegacyDec{}, fmt.Errorf("no liquidity for: %v", denomTo)
	}

	amount := ConstantProductTrade(poolFrom, poolTo, offer)
	feeAmount := amount.Mul(fee)
	amount = amount.Sub(feeAmount)
	return amount, nil
}

func (k Keeper) validateTradeOptions(ctx *types.TradeContext) error {
	if ctx.CoinSource == "" {
		return types.ErrNoCoinSourceGiven
	}

	if ctx.CoinTarget == "" {
		return types.ErrNoCoinTargetGiven
	}

	if ctx.TradeDenomStart == ctx.TradeDenomEnd {
		return types.ErrSameDenom
	}

	if ctx.GivenAmount.Equal(math.ZeroInt()) {
		return types.ErrZeroAmount
	}

	if ctx.GivenAmount.LT(math.ZeroInt()) {
		return types.ErrNegativeAmount
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomStart) {
		return types.ErrDenomNotFound
	}

	if !k.DenomKeeper.IsValidDenom(ctx, ctx.TradeDenomEnd) {
		return types.ErrDenomNotFound
	}

	if ctx.TradeCalculation == nil {
		ctx.TradeCalculation = ConstantProduct{}
	}

	return nil
}
