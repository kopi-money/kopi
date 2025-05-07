package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SumLiquidity is used in testing
func (k Keeper) SumLiquidity(ctx context.Context, denom string) math.Int {
	liqSum := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Amount.IsNil() || liq.Amount.IsZero() {
			continue
		}

		liqSum = liqSum.Add(liq.Amount)
	}

	return liqSum
}

func (k Keeper) GetPoolLiquidity(ctx context.Context, denom string) math.Int {
	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	return k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress()).AmountOf(denom)
}

func (k Keeper) GetEffectiveLiquidity(ctx context.Context, denom string) math.LegacyDec {
	poolLiquidity := k.GetPoolLiquidity(ctx, denom).ToLegacyDec()
	movingLiquidity := k.getMovingLiquidity(ctx, denom).Amount
	if movingLiquidity.IsZero() {
		return poolLiquidity
	}

	return math.LegacyMinDec(poolLiquidity, movingLiquidity)
}

func (k Keeper) GetEffectiveLiquidityFromCache(ctx types.TradeContext, denom string) math.LegacyDec {
	poolLiquidity := ctx.OrdersCaches.LiquidityPool.Get(denom).ToLegacyDec()
	movingLiquidity := ctx.OrdersCaches.MovingLiquidity.Get(denom)
	if movingLiquidity.IsZero() {
		return poolLiquidity
	}

	return math.LegacyMinDec(poolLiquidity, movingLiquidity)
}

// getEffectiveLiquidity return the effictive trade usable liquidity for a given pair (ie XKP and a second denom).
func (k Keeper) getEffectiveLiquidity(ctx context.Context, denom string) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	tradeValue := k.getMovingLiquidity(ctx, denom).Amount
	minimumLiquidity := k.DenomKeeper.MinLiquidity(ctx, denom).ToLegacyDec()

	if minimumLiquidity.GT(tradeValue) {
		tradeValue = minimumLiquidity
	}

	tradeValueBase, _ := k.DenomKeeper.GetValueInBase(ctx, denom, tradeValue)
	minimumLiquidityBase, _ := k.DenomKeeper.GetValueInBase(ctx, denom, minimumLiquidity)
	return tradeValue, tradeValueBase, minimumLiquidityBase
}

func (k Keeper) getEffectiveLiquidityFromCache(ctx types.TradeContext, denom string) types.TradeLiquidity {
	actual := ctx.OrdersCaches.LiquidityPool.Get(denom).ToLegacyDec()
	tradeValue := ctx.OrdersCaches.MovingLiquidity.Get(denom)
	minimumLiquidity := ctx.OrdersCaches.MinimumLiquidity.Get(denom)

	var virtual math.LegacyDec
	if minimumLiquidity.GT(tradeValue) {
		virtual = minimumLiquidity.Sub(tradeValue)
		tradeValue = minimumLiquidity
	} else {
		virtual = math.LegacyZeroDec()
	}

	tradeValueBase, _ := k.DenomKeeper.GetValueInBase(ctx, denom, tradeValue)
	minimumTradeValueBase, _ := k.DenomKeeper.GetValueInBase(ctx, denom, minimumLiquidity)

	return types.TradeLiquidity{
		Actual:                actual,
		TradeValue:            tradeValue,
		TradeValueBase:        tradeValueBase,
		MinimumTradeValueBase: minimumTradeValueBase,
		Virtual:               virtual,
	}
}

// getEffectiveLiquidityForAddress returns the effictive trade liquidity for a given address: When, for example, a
// wallet wants to buy kUSD, there are 1000 kUSD liquidity and 100 belong to that wallet, that wallet can buy up to
// 900 kUSD. This is to prevent self-trading, ie wallets buying their own liquidity.
func (k Keeper) getUsableLiquidityForAddress(ctx types.TradeContext, denom, address string) math.LegacyDec {
	poolLiquidity := ctx.OrdersCaches.LiquidityPool.Get(denom).ToLegacyDec()
	addressLiq := ctx.OrdersCaches.GetLiquidityAddressSum(denom, address)
	poolLiquidity = poolLiquidity.Sub(addressLiq.ToLegacyDec())
	return poolLiquidity
}
