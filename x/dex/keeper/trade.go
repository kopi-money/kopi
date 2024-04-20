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
func (k Keeper) ExecuteTrade(ctx context.Context, eventManager sdk.EventManagerI, options types.TradeOptions) (math.Int, math.Int, math.Int, math.Int, error) {
	if err := k.validateTradeOptions(ctx, &options); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "error in trade options")
	}

	// The address executing the trade might not be the one eligible for a discount. For example, the protocol might
	// sell a user's collateral to partially repay a loan. The protocol does not receive discount when trading, but the
	// user being liquidated does.
	if options.DiscountAddress == nil {
		options.DiscountAddress = options.CoinTarget
	}

	tradeFee := k.getTradeFee(ctx, options.TradeDenomStart, options.TradeDenomEnd, options.DiscountAddress.String(), options.ExcludeFromDiscount)

	// When a maximum price is set, it is checked how much has to be given to achieve the maximum price. If that amount
	// is lower than what it is intended to be given, it means trading with the intended amount would result in a higher
	// price than wanted. In that case, the trade amount is either lowered when the user accepts an incomplete trade, or
	// an error is returned.
	if options.MaxPrice != nil {
		priceAmount := k.calculateAmountGivenPrice(ctx, options.TradeDenomStart, options.TradeDenomEnd, *options.MaxPrice, tradeFee).TruncateInt()
		if priceAmount.LTE(math.ZeroInt()) {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrPriceTooLow
		}

		if priceAmount.LT(options.GivenAmount) {
			if options.AllowIncomplete {
				options.GivenAmount = priceAmount
			} else {
				return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrPriceTooLow
			}
		}
	}

	// With the given funds and the liquidity on the DEX, we can calculate how much a user is to receive when trading.
	// In some cases though, caused by virtual liquidity, the user would receive more than there is liquidity present.
	// In those caes, the given amount is lowered if the user is okay with an incomplete trade. If not, an error is
	// returned.
	maximumTradableAmount := k.CalculateMaximumTradableAmount(ctx, options.TradeDenomStart, options.TradeDenomEnd, false)
	if maximumTradableAmount != nil && maximumTradableAmount.LT(options.GivenAmount) {
		if (*maximumTradableAmount).GT(math.ZeroInt()) && options.AllowIncomplete {
			options.GivenAmount = *maximumTradableAmount
		} else {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughLiquidity
		}
	}

	// If the trade amount is too small, an error is returned. The reason for that is that small trade amounts are more
	// affected by rounding issues.
	if options.GivenAmount.LT(math.NewInt(1000)) {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrTradeAmountTooSmall
	}

	// If the user doesn not have enough funds given the trade amount, an error is returned.
	if err := k.checkSpendableCoins(ctx, options.CoinSource, options.TradeDenomStart, options.GivenAmount); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughFunds
	}

	// Additional check whether there is enough liquidity
	if err := k.checkTradePoolLiquidities(ctx, options); err != nil {
		if options.AllowIncomplete {
			return math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), nil
		} else {
			return math.Int{}, math.Int{}, math.Int{}, math.Int{}, types.ErrNotEnoughLiquidity
		}
	}

	// First trade step from the starting currency to the base currency
	tradeOptions1 := options.TradeToBase(tradeFee)
	usedAmount1, amountReceived1, feePaid1, err := k.ExecuteTradeStep(ctx, eventManager, tradeOptions1)
	if err != nil {
		return math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), errors.Wrap(err, "could not execute trade step 1")
	}

	// Reimburse the user for the trade fee
	reimbursement := k.reimburseFee(ctx, feePaid1)
	amountReceived1 = amountReceived1.Add(reimbursement)

	// Second trade from the base currency to the target currency
	tradeOptions2 := options.TradeToTarget(tradeFee, amountReceived1)
	usedAmount2, amountReceived2, feePaid2, err := k.ExecuteTradeStep(ctx, eventManager, tradeOptions2)
	if err != nil {
		return math.Int{}, math.Int{}, math.ZeroInt(), math.ZeroInt(), errors.Wrap(err, "could not execute trade step 2")
	}

	k.updateRatios(ctx, options.TradeDenomStart)
	k.updateRatios(ctx, options.TradeDenomEnd)

	var usedAmount math.Int
	if options.TradeDenomStart != utils.BaseCurrency {
		usedAmount = usedAmount1
	} else {
		usedAmount = usedAmount2
	}

	k.AddTradeAmount(ctx, options.CoinTarget.String(), amountReceived1)

	if feePaid1.GT(math.ZeroInt()) {
		eventManager.EmitEvent(
			sdk.NewEvent("trade_fee_paid",
				sdk.Attribute{Key: "denom", Value: options.TradeDenomStart},
				sdk.Attribute{Key: "amount", Value: feePaid1.String()},
			),
		)
	}

	if feePaid2.GT(math.ZeroInt()) {
		eventManager.EmitEvent(
			sdk.NewEvent("trade_fee_paid",
				sdk.Attribute{Key: "denom", Value: utils.BaseCurrency},
				sdk.Attribute{Key: "amount", Value: feePaid2.String()},
			),
		)
	}

	if reimbursement.GT(math.ZeroInt()) {
		eventManager.EmitEvent(
			sdk.NewEvent("trade_fee_reimbursed",
				sdk.Attribute{Key: "amount", Value: reimbursement.String()},
			),
		)
	}

	eventManager.EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: options.CoinTarget.String()},
			sdk.Attribute{Key: "from", Value: options.TradeDenomStart},
			sdk.Attribute{Key: "to", Value: options.TradeDenomEnd},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: amountReceived1.String()},
			sdk.Attribute{Key: "amount_used", Value: usedAmount.String()},
			sdk.Attribute{Key: "amount_received", Value: amountReceived2.String()},
		),
	)

	return usedAmount, amountReceived2, feePaid1, feePaid2, nil
}

func (k Keeper) calculateAmountGivenPrice(ctx context.Context, denomFrom, denomTo string, maxPrice, fee math.LegacyDec) math.LegacyDec {
	liqFrom := k.GetFullLiquidity(ctx, denomFrom, denomTo)
	liqTo := k.GetFullLiquidity(ctx, denomTo, denomFrom)
	maxPrice = maxPrice.Mul(math.LegacyOneDec().Sub(fee))
	return maxPrice.Mul(liqTo).Sub(liqFrom)
}

func (k Keeper) reimburseFee(ctx context.Context, feePaid math.Int) math.Int {
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

	sendCoins := sdk.NewCoins(sdk.NewCoin(utils.BaseCurrency, reimburse))
	_ = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFees, types.ModuleName, sendCoins)

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

// updateRatios is called after each trade to adjust the ratio to the changed liquidity amounts
func (k Keeper) updateRatios(ctx context.Context, denom string) {
	if denom != utils.BaseCurrency {
		ratio, _ := k.GetRatio(ctx, denom)

		fullBase := k.GetFullLiquidityBase(ctx, denom)
		fullOther := k.GetFullLiquidityOther(ctx, denom)

		if fullBase.GT(math.LegacyZeroDec()) {
			r := fullOther.Quo(fullBase)
			ratio.Denom = denom
			ratio.Ratio = &r
			k.SetRatio(ctx, ratio)
		}
	}
}

func (k Keeper) TradeSimulation(ctx context.Context, denomFrom, denomTo, address string, amountStart math.Int, excludeFromDiscount bool) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	fee := k.getTradeFee(ctx, denomFrom, denomTo, address, excludeFromDiscount)
	return k.simulateTradeWithFee(ctx, denomFrom, denomTo, amountStart, fee)
}

// SimulateTradeForReserve is used when calculating the profitability of a mint/burn trade. When trading, the reserve
// has to pay the trade fee. However, part of it will be paid out to itself. Thus, when estimating the profitability of
// a trade, that part of the fee is removed.
func (k Keeper) SimulateTradeForReserve(ctx context.Context, denomFrom, denomTo string, amountStart math.Int) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	reserveShare := k.GetParams(ctx).ReserveShare
	fee := k.getTradeFee(ctx, denomFrom, denomTo, "", true)
	fee = fee.Mul(math.LegacyOneDec().Sub(reserveShare))
	return k.simulateTradeWithFee(ctx, denomFrom, denomTo, amountStart, fee)
}

func (k Keeper) simulateTradeWithFee(ctx context.Context, denomFrom, denomTo string, amountStartInt math.Int, fee math.LegacyDec) (math.Int, math.LegacyDec, math.LegacyDec, error) {
	if denomFrom == denomTo {
		return amountStartInt, math.LegacyZeroDec(), math.LegacyOneDec(), nil
	}

	if amountStartInt.Equal(math.ZeroInt()) {
		return math.ZeroInt(), math.LegacyZeroDec(), math.LegacyZeroDec(), nil
	}

	if !k.DenomKeeper.IsValidDenom(ctx, denomFrom) || !k.DenomKeeper.IsValidDenom(ctx, denomTo) {
		return math.Int{}, math.LegacyDec{}, math.LegacyDec{}, types.ErrDenomNotFound
	}

	amountStart := amountStartInt.ToLegacyDec()
	feePaid := amountStart.Mul(fee)
	amountStart = amountStart.Sub(feePaid)

	amountReceived, err := k.calculateSingleTrade(ctx, denomFrom, utils.BaseCurrency, amountStart, fee)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, math.LegacyDec{}, errors.Wrap(err, "could not calculate single trade #1")
	}

	amountReceived, err = k.calculateSingleTrade(ctx, utils.BaseCurrency, denomTo, amountReceived, fee)
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
func (k Keeper) CalculateMaximumTradableAmount(ctx context.Context, denomFrom, denomTo string, debug bool) *math.Int {
	var max1, max2 *math.Int
	if denomTo != utils.BaseCurrency {
		max2 = k.CalculateSingleMaximumTradableAmount(ctx, utils.BaseCurrency, denomTo, nil, debug)
	}

	if denomFrom != utils.BaseCurrency {
		max1 = k.CalculateSingleMaximumTradableAmount(ctx, denomFrom, utils.BaseCurrency, max2, debug)
	} else {
		max1 = max2
	}

	return max1
}

// CalculateSingleMaximumTradableAmount calculates the maximum trading amount for a given trading pair, i.e. how much of
// denomFrom can be given at maximum. When there is no virtual liquidity, the tradable amount is infinity, thus the
// return amount is nil.
func (k Keeper) CalculateSingleMaximumTradableAmount(ctx context.Context, denomFrom, denomTo string, maximumActual *math.Int, debug bool) *math.Int {
	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	dexPool := k.BankKeeper.SpendableCoins(ctx, dexAcc.GetAddress())

	actualFromInt := dexPool.AmountOf(denomFrom)
	actualToInt := dexPool.AmountOf(denomTo)

	var virtualFrom, virtualTo math.LegacyDec
	if denomFrom == utils.BaseCurrency {
		pair, _ := k.GetLiquidityPair(ctx, denomTo)
		virtualFrom = pair.VirtualBase
		virtualTo = pair.VirtualOther

	} else {
		pair, _ := k.GetLiquidityPair(ctx, denomFrom)
		virtualTo = pair.VirtualBase
		virtualFrom = pair.VirtualOther
	}

	actualFrom := actualFromInt.ToLegacyDec()
	actualTo := actualToInt.ToLegacyDec()

	if maximumActual != nil && maximumActual.ToLegacyDec().LT(actualTo) {
		virtualTo = actualTo.Add(virtualTo).Sub((*maximumActual).ToLegacyDec())
		actualTo = (*maximumActual).ToLegacyDec()
	}

	if virtualTo.Equal(math.LegacyZeroDec()) {
		return nil
	}

	maximumTradable := CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	if maximumTradable == nil {
		return nil
	}

	maximumTradableInt := maximumTradable.TruncateInt()
	return &maximumTradableInt
}

func CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo math.LegacyDec) *math.LegacyDec {
	if virtualTo.Equal(math.LegacyZeroDec()) {
		return nil
	}

	A := actualFrom.Add(virtualFrom)
	B := actualTo.Add(virtualTo)
	K := A.Mul(B)

	maximum := K.Quo(virtualTo).Sub(A)
	return &maximum
}

func (k Keeper) checkTradePoolLiquidities(ctx context.Context, options types.TradeOptions) error {
	if options.TradeDenomStart != utils.BaseCurrency {
		if err := k.checkPoolLiquidities(ctx, options.TradeDenomStart, utils.BaseCurrency); err != nil {
			return err
		}
	}

	if options.TradeDenomEnd != utils.BaseCurrency {
		if err := k.checkPoolLiquidities(ctx, utils.BaseCurrency, options.TradeDenomEnd); err != nil {
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
func (k Keeper) ExecuteTradeStep(ctx context.Context, eventManager sdk.EventManagerI, options types.TradeStepOptions) (math.Int, math.Int, math.Int, error) {
	// If a trade is from XKP to something else, the following step send the XKP to the module in trade step 1
	if options.StepDenomTo == utils.BaseCurrency && options.TradeDenomStart == utils.BaseCurrency {
		coins := sdk.NewCoins(sdk.NewCoin(utils.BaseCurrency, options.Amount))
		if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, options.CoinSource, types.PoolTrade, coins); err != nil {
			return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from source to dex (1)")
		}

		return options.Amount, options.Amount, math.ZeroInt(), nil
	}

	// If a trade is from something else to XKP, the following step send the XKP to the user in trade step 2
	if options.StepDenomFrom == utils.BaseCurrency && options.TradeDenomEnd == utils.BaseCurrency {
		coins := sdk.NewCoins(sdk.NewCoin(options.StepDenomTo, options.Amount))
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolTrade, options.CoinTarget, coins); err != nil {
			return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from dex to target (1)")
		}

		return options.Amount, options.Amount, math.ZeroInt(), nil
	}

	// calculate how much the trader can receive with this liquidity entry
	poolFrom, poolTo := k.GetFullLiquidityBaseOther(ctx, options.StepDenomFrom, options.StepDenomTo)
	amountToReceive := options.TradeCalculation.Forward(poolFrom, poolTo, options.Amount.ToLegacyDec())
	if amountToReceive.Equal(math.ZeroInt()) {
		return math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), nil
	}

	liquidityProviders, amountToReceiveLeft, err := k.determineLiquidityProviders(ctx, eventManager, amountToReceive, options.StepDenomFrom, options.StepDenomTo)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from source to dex (2)")
	}
	amountReceivedGross := amountToReceive.Sub(amountToReceiveLeft)

	shareUsed := amountReceivedGross.ToLegacyDec().Quo(amountToReceive.ToLegacyDec())
	amountUsed := shareUsed.Mul(options.Amount.ToLegacyDec()).RoundInt()

	if amountUsed.GT(math.ZeroInt()) && options.StepDenomFrom != utils.BaseCurrency {
		coins := sdk.NewCoins(sdk.NewCoin(options.StepDenomFrom, amountUsed))
		if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, options.CoinSource, types.PoolTrade, coins); err != nil {
			return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from source to dex (2)")
		}
	}

	feePaid, feeForReserve, feeForLiquidityProviders := k.manageFee(ctx, amountReceivedGross, options.TradeFee)
	if err = k.distributeFeeForLiquidityProviders(ctx, liquidityProviders, feeForLiquidityProviders, options.StepDenomTo); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not distribute TO funds to liquidity providers")
	}

	if err = k.distributeGivenFunds(ctx, liquidityProviders, amountUsed, options.StepDenomFrom); err != nil {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not distribute FROM funds to liquidity providers")
	}

	if feeForReserve.GT(math.ZeroInt()) {
		coins := sdk.NewCoins(sdk.NewCoin(options.StepDenomTo, feeForReserve))
		if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolTrade, types.PoolReserve, coins); err != nil {
			return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send coins to reserve")
		}
	}

	poolTrade := k.AccountKeeper.GetModuleAccount(ctx, types.PoolTrade)
	payoutAmount := k.BankKeeper.SpendableCoin(ctx, poolTrade.GetAddress(), options.StepDenomTo).Amount

	if payoutAmount.GT(math.ZeroInt()) && options.StepDenomTo != utils.BaseCurrency {
		coins := sdk.NewCoins(sdk.NewCoin(options.StepDenomTo, payoutAmount))
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolTrade, options.CoinTarget, coins); err != nil {
			return math.Int{}, math.Int{}, math.Int{}, errors.Wrap(err, "could not send from dex to target (2)")
		}
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

// calculatePayoutAmount returns the amount that is paid out to a trader after a trade. The trader receives the excess
// funds in the dex pool. It is possible to calculate the payout amount differently, however, due to rounding those
// values might differ which causes issues long-term. Doing it this way always returns the exact available amount.
func (k Keeper) calculatePayoutAmount(ctx context.Context, tradeAcc sdk.AccAddress, denom string) (math.Int, error) {
	liqSum, _ := k.GetLiquiditySum(ctx, denom)
	dexPool := k.BankKeeper.SpendableCoins(ctx, tradeAcc)
	poolAmount := dexPool.AmountOf(denom)
	payoutAmount := poolAmount.Sub(liqSum)

	//if payoutAmount.LT(math.ZeroInt()) {
	//	return math.Int{}, fmt.Errorf("negative payout. pool amount: %v, liqSum: %v", poolAmount.String(), liqSum.String())
	//}

	return payoutAmount, nil
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

func (k Keeper) validateTradeOptions(ctx context.Context, options *types.TradeOptions) error {
	if options.CoinSource == nil {
		return types.ErrNoCoinSourceGiven
	}

	if options.CoinTarget == nil {
		return types.ErrNoCoinTargetGiven
	}

	if options.TradeDenomStart == options.TradeDenomEnd {
		return types.ErrSameDenom
	}

	if options.GivenAmount.Equal(math.ZeroInt()) {
		return types.ErrZeroAmount
	}

	if options.GivenAmount.LT(math.ZeroInt()) {
		return types.ErrNegativeAmount
	}

	if !k.DenomKeeper.IsValidDenom(ctx, options.TradeDenomStart) {
		return types.ErrDenomNotFound
	}

	if !k.DenomKeeper.IsValidDenom(ctx, options.TradeDenomEnd) {
		return types.ErrDenomNotFound
	}

	if options.TradeCalculation == nil {
		options.TradeCalculation = ConstantProduct{}
	}

	return nil
}
