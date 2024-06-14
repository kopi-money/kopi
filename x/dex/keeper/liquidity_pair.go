package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetLiquidityPair(ctx context.Context, denom string) (types.LiquidityPair, error) {
	ratio, err := k.GetRatio(ctx, denom)
	if err != nil {
		return types.LiquidityPair{}, err
	}

	return k.CreateLiquidityPair(ctx, ratio), nil
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		pair, _ := k.GetLiquidityPair(ctx, denom)
		list = append(list, pair)
	}

	return
}

func (k Keeper) calcVirtualAmountOther(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqBaseDec.Mul(ratio).Sub(liqOtherDec)
}

func (k Keeper) calcVirtualAmountBase(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqOtherDec.Quo(ratio).Sub(liqBaseDec)
}

func (k Keeper) CreateLiquidityPair(ctx context.Context, ratio types.Ratio) (pair types.LiquidityPair) {
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, ratio.Denom)

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
	if denom == utils.BaseCurrency {
		return ordersCaches.FullLiquidityBase.Get(other)
	} else {
		return ordersCaches.FullLiquidityOther.Get(denom)
	}
}

func (k Keeper) GetFullLiquidityBaseOther(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, math.LegacyDec) {
	var liq1, liq2 math.LegacyDec

	if denomFrom == utils.BaseCurrency {
		liq1 = k.GetFullLiquidityBase(ctx, denomTo)
		liq2 = k.GetFullLiquidityOther(ctx, denomTo)
	} else {
		liq1 = k.GetFullLiquidityOther(ctx, denomFrom)
		liq2 = k.GetFullLiquidityBase(ctx, denomFrom)
	}

	return liq1, liq2
}

func (k Keeper) GetFullLiquidityBase(ctx context.Context, denomOther string) math.LegacyDec {
	if denomOther == utils.BaseCurrency {
		panic("other denom cannot be base currency")
	}

	liq1 := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liq2, _ := k.GetLiquidityPair(ctx, denomOther)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualBase)
}

func (k Keeper) GetFullLiquidityOther(ctx context.Context, denom string) math.LegacyDec {
	liq1 := k.GetLiquiditySum(ctx, denom)
	liq2, _ := k.GetLiquidityPair(ctx, denom)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualOther)
}

func sumLiquidity(actual, virtual math.LegacyDec) math.LegacyDec {
	if actual.IsNil() {
		panic("actual liquidity is nil")
	}
	if virtual.IsNil() {
		virtual = math.LegacyZeroDec()
	}

	return actual.Add(virtual)
}

func compareLiquidityPairs(lp1, lp2 types.LiquidityPair) bool {
	return lp1.Denom == lp2.Denom &&
		lp1.VirtualBase.Equal(lp2.VirtualBase) &&
		lp1.VirtualOther.Equal(lp2.VirtualOther)
}
