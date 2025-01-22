package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetLiquidityPair(ctx context.Context, denom string) (types.LiquidityPair, error) {
	ratio, err := k.DenomKeeper.GetRatio(ctx, denom)
	if err != nil {
		return types.LiquidityPair{}, err
	}

	return k.CreateLiquidityPair(ctx, ratio), nil
}

func (k Keeper) GetLiquidityPairWithLiquidity(ctx context.Context, denom string, liqBase, liqOther math.Int, sizeFactor math.LegacyDec) (types.LiquidityPair, error) {
	ratio, err := k.DenomKeeper.GetRatio(ctx, denom)
	if err != nil {
		return types.LiquidityPair{}, err
	}

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, sizeFactor), nil
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		pair, _ := k.GetLiquidityPair(ctx, denom)
		list = append(list, pair)
	}

	return
}

func (k Keeper) CreateLiquidityPair(ctx context.Context, ratio denomtypes.Ratio) (pair types.LiquidityPair) {
	liqBase := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, math.LegacyOneDec())
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ctx context.Context, ratio denomtypes.Ratio, liqBase, liqOther math.Int, sizeFactor math.LegacyDec) (pair types.LiquidityPair) {
	liqBaseDec := liqBase.ToLegacyDec().Mul(sizeFactor)
	liqOtherDec := liqOther.ToLegacyDec().Mul(sizeFactor)

	pair.Denom = ratio.Denom

	if liqBaseDec.Mul(ratio.Ratio).GT(liqOtherDec) {
		pair.VirtualBase = math.LegacyZeroDec()
		pair.VirtualOther = liqBaseDec.Mul(ratio.Ratio).Sub(liqOtherDec)
	} else {
		pair.VirtualBase = liqOtherDec.Quo(ratio.Ratio).Sub(liqBaseDec)
		pair.VirtualOther = math.LegacyZeroDec()
	}

	return
}

func (k Keeper) GetFullLiquidity(ordersCaches *types.OrdersCaches, additionalLiquidity types.AdditionalLiquidity, denom, other string) math.LegacyDec {
	var actual, virtual math.LegacyDec

	if denom == constants.BaseCurrency {
		actual = ordersCaches.LiquidityPool.Get().AmountOf(constants.BaseCurrency).ToLegacyDec()
		pair := ordersCaches.LiquidityPair.Get(other, additionalLiquidity.GetSizeFactor(other))
		virtual = pair.VirtualBase
	} else {
		actual = ordersCaches.LiquidityPool.Get().AmountOf(denom).ToLegacyDec()
		pair := ordersCaches.LiquidityPair.Get(denom, additionalLiquidity.GetSizeFactor(denom))
		virtual = pair.VirtualOther
	}

	return sumLiquidity(actual, virtual)
}

func (k Keeper) GetCrossLiquidity(ctx *types.TradeContext) (math.LegacyDec, math.LegacyDec) {
	fullFrom := GetFullLiquidityOtherCache(ctx.OrdersCaches, ctx.AdditionalLiquidity, ctx.TradeDenomGiving)
	fullFrom = ctx.AdditionalLiquidity.Add(ctx.TradeDenomGiving, fullFrom)

	fullTo := GetFullLiquidityOtherCache(ctx.OrdersCaches, ctx.AdditionalLiquidity, ctx.TradeDenomReceiving)
	fullTo = ctx.AdditionalLiquidity.Add(ctx.TradeDenomReceiving, fullTo)

	return fullFrom, fullTo
}

func (k Keeper) GetFullLiquidityBaseOther(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, math.LegacyDec) {
	var liq1, liq2 math.LegacyDec

	if denomFrom == constants.BaseCurrency {
		liq1 = k.GetFullLiquidityBase(ctx, denomTo)
		liq2 = k.GetFullLiquidityOther(ctx, denomTo)
	} else {
		liq1 = k.GetFullLiquidityOther(ctx, denomFrom)
		liq2 = k.GetFullLiquidityBase(ctx, denomFrom)
	}

	return liq1, liq2
}

func (k Keeper) GetFullLiquidityBase(ctx context.Context, denomOther string) math.LegacyDec {
	if denomOther == constants.BaseCurrency {
		panic("other denom cannot be base currency")
	}

	liq1 := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liq2, _ := k.GetLiquidityPair(ctx, denomOther)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualBase)
}

func (k Keeper) GetFullLiquidityOther(ctx context.Context, denom string) math.LegacyDec {
	liq1 := k.GetLiquiditySum(ctx, denom)
	liq2, _ := k.GetLiquidityPair(ctx, denom)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualOther)
}

func (k Keeper) GetFullLiquidityBaseOtherCache(ordersCache *types.OrdersCaches, additionalLiquidity types.AdditionalLiquidity, denomFrom, denomTo string) (math.LegacyDec, math.LegacyDec) {
	var liq1, liq2 math.LegacyDec

	if denomFrom == constants.BaseCurrency {
		liq1 = GetFullLiquidityBaseCache(ordersCache, additionalLiquidity, denomTo)
		liq2 = GetFullLiquidityOtherCache(ordersCache, additionalLiquidity, denomTo)
	} else {
		liq1 = GetFullLiquidityOtherCache(ordersCache, additionalLiquidity, denomFrom)
		liq2 = GetFullLiquidityBaseCache(ordersCache, additionalLiquidity, denomFrom)
	}

	return liq1, liq2
}

func GetFullLiquidityBaseCache(ordersCache *types.OrdersCaches, additionalLiquidity types.AdditionalLiquidity, other string) math.LegacyDec {
	if other == constants.BaseCurrency {
		panic("other denom cannot be base currency")
	}

	liq1 := ordersCache.LiquidityPool.Get().AmountOf(constants.BaseCurrency)
	sizeFactor := additionalLiquidity.GetSizeFactor(other)
	pair := ordersCache.LiquidityPair.Load(other, sizeFactor)
	return sumLiquidity(liq1.ToLegacyDec(), pair.VirtualBase)
}

func GetFullLiquidityOtherCache(ordersCache *types.OrdersCaches, additionalLiquidity types.AdditionalLiquidity, other string) math.LegacyDec {
	liq1 := ordersCache.LiquidityPool.Get().AmountOf(other)
	sizeFactor := additionalLiquidity.GetSizeFactor(other)
	pair := ordersCache.LiquidityPair.Load(other, sizeFactor)
	return sumLiquidity(liq1.ToLegacyDec(), pair.VirtualOther)
}

func sumLiquidity(actual, virtual math.LegacyDec) math.LegacyDec {
	if actual.IsNil() {
		panic("actual liquidity is nil")
	}
	if virtual.IsNil() || virtual.IsZero() {
		return actual
	}

	return actual.Add(virtual)
}
