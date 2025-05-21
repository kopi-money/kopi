package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/trading"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) ExecuteSell(ctx types.TradeContext) (trading.TradeResult, error) {
	tradeResult, _, err := k.executeSell(ctx)
	return tradeResult, err
}

func (k Keeper) executeSell(ctx types.TradeContext) (trading.TradeResult, math.Int, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeSell
	ctx.CalcMaximumTradableAmountByLiquidity = trading.CalculateMaximumSellableByLiquidity
	ctx.CalcMaximumTradableAmountByPrice = trading.CalculateMaximumSellableByPrice
	ctx.CalcMaximumTradableAmountByWallet = func(_, _ trading.Liquidity, _ math.LegacyDec) math.Int {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		amount := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount

		if ctx.MaximumAvailableAmount.IsNil() {
			return amount
		}

		return math.MinInt(amount, ctx.MaximumAvailableAmount)
	}

	tradeResult, amountIntermediate, err := k.executeTrade(ctx, trading.SellCallbacks())
	if err != nil {
		return trading.TradeResult{}, math.Int{}, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: ctx.CoinTarget},
			sdk.Attribute{Key: "denom_giving", Value: ctx.TradeDenomGiving},
			sdk.Attribute{Key: "denom_receiving", Value: ctx.TradeDenomReceiving},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: amountIntermediate.String()},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven().String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived().String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(ctx.ProtocolTrade)},
		),
	)

	return tradeResult, amountIntermediate, nil
}

func (k Keeper) ExecuteBuy(ctx types.TradeContext) (trading.TradeResult, error) {
	tradeResult, _, err := k.executeBuy(ctx)
	return tradeResult, err
}

func (k Keeper) executeBuy(ctx types.TradeContext) (trading.TradeResult, math.Int, error) {
	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	ctx.TradeType = types.TradeTypeBuy
	ctx.CalcMaximumTradableAmountByLiquidity = trading.CalculateMaximumBuyableByLiquidity
	ctx.CalcMaximumTradableAmountByPrice = trading.CalculateMaximumBuyableByPrice
	ctx.CalcMaximumTradableAmountByWallet = func(liqFrom, liqTo trading.Liquidity, fee math.LegacyDec) math.Int {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		available := k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount

		if !ctx.MaximumAvailableAmount.IsNil() {
			available = math.MinInt(available, ctx.MaximumAvailableAmount)
		}

		feeAmount := available.ToLegacyDec().Mul(fee).TruncateInt()
		available = available.Sub(feeAmount)
		return trading.CalculateMaximumBuyableByWallet(liqFrom, liqTo, available)
	}

	tradeResult, amountIntermediate, err := k.executeTrade(ctx, trading.BuyCallbacks())
	if err != nil {
		return trading.TradeResult{}, math.Int{}, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: ctx.CoinTarget},
			sdk.Attribute{Key: "denom_giving", Value: ctx.TradeDenomGiving},
			sdk.Attribute{Key: "denom_receiving", Value: ctx.TradeDenomReceiving},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: amountIntermediate.String()},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven().String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived().String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(ctx.ProtocolTrade)},
		),
	)

	return tradeResult, amountIntermediate, nil
}

func (k Keeper) executeTrade(ctx types.TradeContext, callbacks trading.Callbacks) (trading.TradeResult, math.Int, error) {
	if err := k.validateTradeOptions(ctx); err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("error in trade options: %w", err)
	}

	if ctx.MaximumAvailableAmount.IsNil() {
		acc, _ := sdk.AccAddressFromBech32(ctx.CoinSource)
		ctx.MaximumAvailableAmount = k.BankKeeper.SpendableCoin(ctx, acc, ctx.TradeDenomGiving).Amount
	}

	// The address executing the trade might not be the one eligible for a discount. For example, the protocol might
	// sell a user's collateral to partially repay a loan. The protocol does not receive a discount when trading, but
	// the user being liquidated does.
	if ctx.DiscountAddress == "" {
		ctx.DiscountAddress = ctx.CoinTarget
	}

	if ctx.OrdersCaches == nil {
		ctx.OrdersCaches = k.NewOrdersCaches(ctx)
	}

	if ctx.Fee == nil {
		discountLevels := ctx.OrdersCaches.DiscountLevels.Get()
		tradeFee := ctx.OrdersCaches.TradeFee.Get()
		tradeFee = k.getDiscountedTradeFee(ctx, discountLevels, tradeFee, ctx.DiscountAddress, ctx.ExcludeFromDiscount)
		ctx.Fee = &tradeFee
	}

	// The liquidity amounts used for this trade. When there just has been new liquidity added to one of the denoms,
	// not of all of it will be used right away. For example, when there are 10k kUSD and 10k USDC but of the 10k USDC
	// 9k have just been added, then the two liquidity amounts will be around 1k, thereby resulting in a higher spread.
	liqFrom, liqTo, tradeValue, err := k.CalculateTradeLiquidityFromCache(ctx, ctx.TradeDenomGiving, ctx.TradeDenomReceiving)
	if err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("trade liquidity: %w", err)
	}

	// When selling:
	// With the given funds and the liquidity on the DEX, we can calculate how much a user is to receive when trading.
	// In some cases though, caused by virtual liquidity, the user would receive more than there is liquidity present.
	// In those cases, the given amount is lowered if the user is okay with an incomplete trade. If not, an error is
	// returned.
	// When buying:
	// Given how much funds are in the user's wallet, the user might not get the full desired amount.
	maximumTradableAmount := ctx.CalcMaximumTradableAmountByLiquidity(liqFrom, liqTo)
	if maximumTradableAmount != nil && maximumTradableAmount.LT(ctx.TradeAmount) {
		if ctx.MinimumTradeAmount != nil && maximumTradableAmount.LT(*ctx.MinimumTradeAmount) {
			return trading.TradeResult{}, math.Int{}, types.ErrNotEnoughBuyableLiquidity
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
		priceAmount := ctx.CalcMaximumTradableAmountByPrice(liqFrom.Full(), liqTo.Full(), ctx.MaxPrice.MaxPrice).TruncateInt()
		if !priceAmount.IsPositive() {
			return trading.TradeResult{}, math.Int{}, types.ErrNegativeTradeAmount
		}

		if priceAmount.LT(ctx.TradeAmount) {
			if ctx.MinimumTradeAmount != nil && !ctx.MinimumTradeAmount.IsNil() && ctx.MinimumTradeAmount.GT(priceAmount) {
				return trading.TradeResult{}, math.Int{}, trading.ErrMarketPriceTooHigh
			}

			ctx.TradeAmount = priceAmount
		}
	}

	// When selling:
	// The available sellable amount might be smaller than what's in the wallet: For example, When the protocol is
	// executing an order or is selling collateral, that protocol address has funds for many other wallets but the
	// available amount for trading is determined by other factors.
	// When buying:
	// Given how much funds are in the user's wallet, the user might not get the full desired amount.
	availableByWallet := ctx.CalcMaximumTradableAmountByWallet(liqFrom, liqTo, *ctx.Fee)
	if availableByWallet.LT(ctx.TradeAmount) {
		if ctx.MinimumTradeAmount != nil && availableByWallet.LT(*ctx.MinimumTradeAmount) {
			return trading.TradeResult{}, math.Int{}, types.ErrNotEnoughFunds
		}

		ctx.TradeAmount = availableByWallet
	}

	tradeData := trading.TradeData{
		MaxPrice:           ctx.MaxPrice,
		TradeAmount:        ctx.TradeAmount,
		MinimumTradeAmount: ctx.MinimumTradeAmount,
		LiqFrom:            liqFrom,
		LiqTo:              liqTo,
		Fee:                *ctx.Fee,
		Callbacks:          callbacks,
	}

	if ctx.FlatPrice != nil {
		tradeData.CPTrade = ctx.FlatPrice
	}

	// Do the actual trade, ie calculating how much the user has to pay (when buying) or how much the user will receive
	// (when selling).
	tradeResult, err := trading.Trade(tradeData)
	if err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("execute trade: %w", err)
	}

	if !tradeResult.AmountGivenNet().IsPositive() {
		return trading.TradeResult{}, math.Int{}, types.ErrZeroTrade
	}

	if coinTarget := ctx.CoinTargetForEffective(); coinTarget != nil {
		usable := k.getUsableLiquidityForAddress(ctx, ctx.TradeDenomReceiving, *coinTarget)
		if usable.TruncateInt().LT(tradeResult.AmountReceivedGross()) {
			return trading.TradeResult{}, math.Int{}, types.ErrNotEnoughUsableLiquidity
		}
	}

	// Remove bought liquidity from the DEX and add sold funds as new liquidity.
	changeFrom, changeTo, err := k.HandleLiquidity(ctx, tradeResult, callbacks.GetFee)
	if err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("handle liquidity: %w", err)
	}

	// Update prices
	if err = k.UpdateRatiosToBase(ctx, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, tradeValue, changeFrom, changeTo); err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("update ratios to base: %w", err)
	}

	// For statistics
	amountBase, err := k.tradeAmountToBase(ctx, callbacks.GetTradeDenom, tradeResult.TradeAmount)
	if err != nil {
		return trading.TradeResult{}, math.Int{}, fmt.Errorf("convert trade amount to base: %w", err)
	}

	ctx.OrdersCaches.Clear()
	k.AddTradeAmount(ctx, ctx.CoinTarget, amountBase)

	return tradeResult, amountBase, nil
}

func (k Keeper) getDiscountedTradeFee(ctx context.Context, discountLevels []types.DiscountLevel, fee math.LegacyDec, discountAddress string, excludeFromDiscount bool) math.LegacyDec {
	discount := k.getTradeDiscount(ctx, discountLevels, discountAddress, excludeFromDiscount)
	discount = math.LegacyOneDec().Sub(discount)
	fee = fee.Mul(discount)

	return fee
}

func (k Keeper) tradeAmountToBase(ctx types.TradeContext, getTradeDenom trading.GetTradeDenom, tradeAmount math.Int) (math.Int, error) {
	amount, err := k.DenomKeeper.GetValueInBase(ctx, getTradeDenom(ctx.TradeDenomGiving, ctx.TradeDenomReceiving), tradeAmount.ToLegacyDec())
	if err != nil {
		return math.Int{}, err
	}

	return amount.RoundInt(), nil
}

// HandleLiquidity first removes the bought liquidity from the exchange and determines which liquidity positions will
// be used. Then trade fees are distributed before added the sold funds as new liquidity to the DEX.
func (k Keeper) HandleLiquidity(ctx types.TradeContext, tradeResult trading.TradeResult, getFee trading.GetFeeAmount) (math.Int, math.Int, error) {
	accPoolTrade := ctx.OrdersCaches.AccPoolTrade.Get().String()
	accPoolReserve := ctx.OrdersCaches.AccPoolReserve.Get().String()
	accPoolFeeIncome := ctx.OrdersCaches.AccPoolFeeIncome.Get().String()

	poolFrom1 := ctx.OrdersCaches.LiquidityPool.Get(ctx.TradeDenomGiving)
	poolTo1 := ctx.OrdersCaches.LiquidityPool.Get(ctx.TradeDenomReceiving)

	liquidityProviders, amountToReceiveLeft, err := k.determineLiquidityProviders(ctx, tradeResult.AmountReceivedGross(), ctx.TradeDenomReceiving, ctx.CoinTarget, ctx.ProtocolTrade)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("send from source to dex (2): %w", err)
	}

	if amountToReceiveLeft.IsPositive() {
		return math.Int{}, math.Int{}, fmt.Errorf("fulfill trade")
	}

	feeForReserve, feeForLiquidityProviders := manageFee(getFee(tradeResult), ctx.OrdersCaches.ReserveFeeShare.Get())
	if feeForLiquidityProviders.IsPositive() {
		ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolFeeIncome, ctx.FeeDenom(), feeForLiquidityProviders)
	}

	if err = k.distributeGivenFunds(ctx, ctx.OrdersCaches, liquidityProviders, tradeResult.AmountGivenNet(), tradeResult.AmountReceived(), ctx.TradeDenomGiving); err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("distribute FROM funds to liquidity providers: %w", err)
	}

	ctx.TradeBalances.AddTransfer(accPoolTrade, accPoolReserve, ctx.FeeDenom(), feeForReserve)
	ctx.TradeBalances.AddTransfer(ctx.CoinSource, accPoolTrade, ctx.TradeDenomGiving, tradeResult.AmountGiven())

	payoutAmount := ctx.TradeBalances.NetBalance(accPoolTrade, ctx.TradeDenomReceiving)
	ctx.TradeBalances.AddTransfer(accPoolTrade, ctx.CoinTarget, ctx.TradeDenomReceiving, payoutAmount)

	poolFrom2 := ctx.OrdersCaches.LiquidityPool.Get(ctx.TradeDenomGiving)
	poolTo2 := ctx.OrdersCaches.LiquidityPool.Get(ctx.TradeDenomReceiving)
	changeFrom := poolFrom2.Sub(poolFrom1)
	changeTo := poolTo2.Sub(poolTo1)

	return changeFrom, changeTo, nil
}

func (k Keeper) UpdateRatiosToBase(ctx context.Context, tradeDenomGiving, tradeDenomReceiving string, tradeValue math.LegacyDec, changeFrom, changeTo math.Int) error {
	liqBase := k.GetEffectiveLiquidity(ctx, constants.BaseCurrency)

	changes := types.AmountsMap{}
	changes.Add(tradeDenomGiving, changeFrom.ToLegacyDec())
	changes.Add(tradeDenomReceiving, changeTo.ToLegacyDec())
	changeBase := changes.AmountOf(constants.BaseCurrency)

	changeRatio := changeBase.Abs().Quo(tradeValue)

	for _, ratio := range k.DenomKeeper.GetAllRatios(ctx) {
		changeOther := changes.AmountOf(ratio.Denom)
		if err := k.UpdateRatioToBase(ctx, ratio, liqBase, changeRatio, changeBase, changeOther); err != nil {
			return fmt.Errorf("update ratio to base (%v): %w", ratio.Denom, err)
		}
	}

	k.updateMovingLiquidityFromTrade(ctx, constants.BaseCurrency, changeBase)

	return nil
}

func (k Keeper) UpdateRatioToBase(ctx context.Context, ratio denomtypes.Ratio, liqBase, changeRatio, changeBase, changeOther math.LegacyDec) error {
	if changeOther.IsZero() && changeBase.IsZero() {
		return nil
	}

	pair, err := k.CreateRatioUpdatePair(ctx, ratio, liqBase)
	if err != nil {
		return err
	}

	if changeRatio.IsPositive() {
		changeRatioThis := changeBase.Abs().Quo(pair.Base.GetFull())
		if changeRatioThis.GT(changeRatio) {
			tradeValueScaling := changeRatio.Quo(changeRatioThis)
			changeBase = changeBase.Mul(tradeValueScaling)
			changeOther = changeOther.Mul(tradeValueScaling)
		}
	}

	fullBase := pair.Base.GetFull().Add(changeBase)
	fullOther := pair.Other.GetFull().Add(changeOther)
	k.updateMovingLiquidityFromTrade(ctx, ratio.Denom, changeOther)

	if fullBase.IsPositive() {
		newRatio := fullOther.Quo(fullBase) // C
		if newRatio.IsPositive() {
			k.DenomKeeper.SetRatio(ctx, denomtypes.Ratio{
				Denom: ratio.Denom,
				Ratio: newRatio,
			})
		}
	}

	return nil
}

func (k Keeper) CreateRatioUpdatePair(ctx context.Context, ratio denomtypes.Ratio, liqBase math.LegacyDec) (types.LiquidityPair, error) {
	liqOther := k.GetEffectiveLiquidity(ctx, ratio.Denom)
	minLiqBase := k.DenomKeeper.MinTradeLiquidity(ctx, constants.BaseCurrency)
	minLiqOther := k.DenomKeeper.MinTradeLiquidity(ctx, ratio.Denom)

	pair, err := k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, minLiqBase, minLiqOther)
	if err != nil {
		return types.LiquidityPair{}, fmt.Errorf("liquidity pair %v: %w", ratio.Denom, err)
	}

	return pair, nil
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

func (k Keeper) addProviderFee(ctx context.Context, amount, tradeFee math.LegacyDec) math.LegacyDec {
	feeShareReserve := k.GetReserveFeeShare(ctx)
	feeShareProvider := math.LegacyOneDec().Sub(feeShareReserve)
	feeProvider := tradeFee.Mul(feeShareProvider)

	return amount.Quo(math.LegacyOneDec().Sub(feeProvider)) // C
}

// manageFee is called each time liquidity is used for a trade. amount indicates how much is traded right now, the
// address corresponds to the user whose liquidity is used right now.
func manageFee(feeAmount math.Int, reserveFeeShare math.LegacyDec) (math.Int, math.Int) {
	feeForReserve := feeAmount.ToLegacyDec().Mul(reserveFeeShare).RoundInt()
	feeForLiquidityProviders := feeAmount.Sub(feeForReserve)
	return feeForReserve, feeForLiquidityProviders
}

func (k Keeper) SimulateSellFromTradeContext(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	return k.Simulate(ctx, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, trading.TradeData{
		Callbacks:   trading.SellCallbacks(),
		TradeAmount: ctx.TradeAmount,
		Fee:         *ctx.Fee,
		MaxPrice:    ctx.MaxPrice,
	})
}

func (k Keeper) SimulateSell(ctx context.Context, tradeAmount math.Int, tradeDenomGiving, tradeDenomReceiving string) (types.TradeSimulationResult, error) {
	return k.Simulate(ctx, tradeDenomGiving, tradeDenomReceiving, trading.TradeData{
		Callbacks:   trading.SellCallbacks(),
		TradeAmount: tradeAmount,
	})
}

func (k Keeper) SimulateBuyFromTradeContext(ctx types.TradeContext) (types.TradeSimulationResult, error) {
	return k.Simulate(ctx, ctx.TradeDenomGiving, ctx.TradeDenomReceiving, trading.TradeData{
		Callbacks:   trading.BuyCallbacks(),
		TradeAmount: ctx.TradeAmount,
		Fee:         *ctx.Fee,
		MaxPrice:    ctx.MaxPrice,
	})
}

func (k Keeper) SimulateBuy(ctx context.Context, tradeAmount math.Int, tradeDenomGiving, tradeDenomReceiving string) (types.TradeSimulationResult, error) {
	return k.Simulate(ctx, tradeDenomGiving, tradeDenomReceiving, trading.TradeData{
		Callbacks:   trading.BuyCallbacks(),
		TradeAmount: tradeAmount,
	})
}

func (k Keeper) Simulate(ctx context.Context, tradeDenomGiving, tradeDenomReceiving string, tradeData trading.TradeData) (types.TradeSimulationResult, error) {
	if tradeData.TradeAmount.IsZero() {
		return types.TradeSimulationResult{
			AmountIntermediate: math.ZeroInt(),
			AmountGiven:        math.ZeroInt(),
			AmountReceived:     math.ZeroInt(),
			FeeGiven:           math.ZeroInt(),
		}, nil
	}

	if !k.DenomKeeper.IsValidDenom(ctx, tradeDenomGiving) || !k.DenomKeeper.IsValidDenom(ctx, tradeDenomReceiving) {
		return types.TradeSimulationResult{}, denomtypes.ErrInvalidDexAsset
	}

	var err error
	tradeData.LiqFrom, tradeData.LiqTo, err = k.CalculateTradeLiquidity(ctx, tradeDenomGiving, tradeDenomReceiving)
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("trade liquidity: %w", err)
	}

	if tradeData.Fee.IsNil() {
		tradeData.Fee = k.GetParams(ctx).TradeFee
	}

	tradeData.TradeAmount, err = trading.HandleMaxPrice(tradeData, trading.KeepMaxPrice)
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("handle max price: %w", err)
	}

	tradeResult, err := trading.Trade(tradeData)
	if err != nil {
		return types.TradeSimulationResult{}, fmt.Errorf("execute trade: %w", err)
	}

	return types.TradeSimulationResult{
		AmountGiven:    tradeResult.AmountGiven(),
		AmountReceived: tradeResult.AmountReceived(),
		FeeGiven:       tradeData.GetFee(tradeResult),
	}, nil
}

func (k Keeper) validateTradeOptions(ctx types.TradeContext) error {
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
