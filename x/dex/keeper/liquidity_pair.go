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

func (k Keeper) GetLiquidityPairWithLiquidity(ctx context.Context, denom string, liqBase, liqOther math.Int) (types.LiquidityPair, error) {
	ratio, err := k.DenomKeeper.GetRatio(ctx, denom)
	if err != nil {
		return types.LiquidityPair{}, err
	}

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther), nil
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		pair, _ := k.GetLiquidityPair(ctx, denom)
		list = append(list, pair)
	}

	return
}

func (k Keeper) calcVirtualAmountOther(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	liqBase := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqBaseDec.Mul(ratio).Sub(liqOtherDec)
}

func (k Keeper) calcVirtualAmountBase(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	if !ratio.IsPositive() {
		panic("ratio must be positive")
	}

	liqBase := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqOtherDec.Quo(ratio).Sub(liqBaseDec)
}

func (k Keeper) CreateLiquidityPair(ctx context.Context, ratio denomtypes.Ratio) (pair types.LiquidityPair) {
	liqBase := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther)
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ctx context.Context, ratio denomtypes.Ratio, liqBase, liqOther math.Int) (pair types.LiquidityPair) {
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	pair.Denom = ratio.Denom

	if liqBaseDec.Mul(ratio.Ratio).GT(liqOtherDec) {
		pair.VirtualBase = math.LegacyZeroDec()
		pair.VirtualOther = k.calcVirtualAmountOther(ctx, ratio.Ratio, ratio.Denom)
	} else {
		pair.VirtualBase = k.calcVirtualAmountBase(ctx, ratio.Ratio, ratio.Denom)
		pair.VirtualOther = math.LegacyZeroDec()
	}

	return
}

func (k Keeper) GetFullLiquidity(ordersCaches *types.OrdersCaches, denom, other string) math.LegacyDec {
	var actual, virtual math.LegacyDec

	if denom == constants.BaseCurrency {
		actual = ordersCaches.LiquidityPool.Get().AmountOf(constants.BaseCurrency).ToLegacyDec()
		pair := ordersCaches.LiquidityPair.Get(other)
		virtual = pair.VirtualBase
	} else {
		actual = ordersCaches.LiquidityPool.Get().AmountOf(denom).ToLegacyDec()
		pair := ordersCaches.LiquidityPair.Get(denom)
		virtual = pair.VirtualOther
	}

	return sumLiquidity(actual, virtual)
}

func (k Keeper) GetCrossLiquidity(ctx *types.TradeContext) (math.LegacyDec, math.LegacyDec) {
	if ctx.TradeDenomGiving == constants.BaseCurrency || ctx.TradeDenomReceiving == constants.BaseCurrency {
		liqFrom := k.GetFullLiquidity(ctx.OrdersCaches, ctx.TradeDenomGiving, ctx.TradeDenomReceiving)
		liqTo := k.GetFullLiquidity(ctx.OrdersCaches, ctx.TradeDenomReceiving, ctx.TradeDenomGiving)
		return liqFrom, liqTo
	}

	fromRatio, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomGiving)
	fromLiquidity := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomGiving).ToLegacyDec()
	fromLiquidityValueBase := fromLiquidity.Quo(fromRatio.Ratio)

	toRatio, _ := k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomReceiving)
	toLiquidity := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving).ToLegacyDec()
	toLiquidityValueBase := toLiquidity.Quo(toRatio.Ratio)

	switch {
	case fromLiquidityValueBase.LT(toLiquidityValueBase):
		missing := toLiquidityValueBase.Sub(fromLiquidityValueBase)
		fromLiquidity = fromLiquidity.Add(missing.Mul(fromRatio.Ratio))
	case toLiquidityValueBase.LT(fromLiquidityValueBase):
		missing := fromLiquidityValueBase.Sub(toLiquidityValueBase)
		toLiquidity = toLiquidity.Add(missing.Mul(toRatio.Ratio))
	}

	return fromLiquidity, toLiquidity
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

func (k Keeper) GetFullLiquidityBaseOtherCache(ordersCache *types.OrdersCaches, denomFrom, denomTo string) (math.LegacyDec, math.LegacyDec) {
	var liq1, liq2 math.LegacyDec

	if denomFrom == constants.BaseCurrency {
		liq1 = k.GetFullLiquidityBaseCache(ordersCache, denomTo)
		liq2 = k.GetFullLiquidityOtherCache(ordersCache, denomTo)
	} else {
		liq1 = k.GetFullLiquidityOtherCache(ordersCache, denomFrom)
		liq2 = k.GetFullLiquidityBaseCache(ordersCache, denomFrom)
	}

	return liq1, liq2
}

func (k Keeper) GetFullLiquidityBaseCache(ordersCache *types.OrdersCaches, other string) math.LegacyDec {
	if other == constants.BaseCurrency {
		panic("other denom cannot be base currency")
	}

	liq1 := ordersCache.LiquidityPool.Get().AmountOf(constants.BaseCurrency)
	pair := ordersCache.LiquidityPair.Get(other)
	return sumLiquidity(liq1.ToLegacyDec(), pair.VirtualBase)
}

func (k Keeper) GetFullLiquidityOtherCache(ordersCache *types.OrdersCaches, other string) math.LegacyDec {
	liq1 := ordersCache.LiquidityPool.Get().AmountOf(other)
	pair := ordersCache.LiquidityPair.Get(other)
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
