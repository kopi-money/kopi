package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SetLiquidityPair set a specific liquidityPair in the store
func (k Keeper) SetLiquidityPair(ctx context.Context, liquidityPair types.LiquidityPair) {
	k.liquidityPairs.Set(ctx, liquidityPair.Denom, liquidityPair)
}

func (k Keeper) GetLiquidityPair(ctx context.Context, denom string) (types.LiquidityPair, bool) {
	return k.liquidityPairs.Get(ctx, denom)
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	iterator := k.liquidityPairs.Iterator(ctx)
	for iterator.Valid() {
		list = append(list, iterator.GetNext())
	}

	return
}

func (k Keeper) InitPairs(ctx context.Context) {
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	if !liqBase.IsNil() {
		for _, denom := range k.DenomKeeper.Denoms(ctx) {
			if denom != utils.BaseCurrency {
				k.initPair(ctx, liqBase, denom)
			}
		}
	}
}

func (k Keeper) initPair(ctx context.Context, liqBase math.Int, denom string) {
	fac := k.DenomKeeper.InitialVirtualLiquidityFactor(ctx, denom)

	pair, found := k.GetLiquidityPair(ctx, denom)
	if !found {
		pair.Denom = denom
		pair.VirtualBase = math.LegacyZeroDec()
		pair.VirtualOther = liqBase.ToLegacyDec().Mul(fac)
		k.SetLiquidityPair(ctx, pair)
	}

	ratio, found := k.GetRatio(ctx, denom)
	if !found || found && ratio.Ratio == nil {
		ratio.Denom = denom
		ratio.Ratio = &fac
		k.SetRatio(ctx, ratio)
	}
}

func (k Keeper) updatePairs(ctx context.Context, ordersCaches *types.OrdersCaches) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != utils.BaseCurrency {
			k.updatePair(ctx, ordersCaches, denom)
		}
	}
}

// updatePair updates the virtual liquidity for a denom pair. This function only is called after having added liqudity
// to a pair.
func (k Keeper) updatePair(ctx context.Context, ordersCaches *types.OrdersCaches, denom string) {
	pair, _ := k.GetLiquidityPair(ctx, denom)
	ratio, found := k.GetRatio(ctx, denom)

	if ratio.Ratio != nil {
		liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		liqOther := k.GetLiquiditySum(ctx, denom)

		liqBaseDec := liqBase.ToLegacyDec()
		liqOtherDec := liqOther.ToLegacyDec()

		if liqBaseDec.Mul(*ratio.Ratio).GT(liqOtherDec) {
			pair.VirtualBase = math.LegacyZeroDec()
			pair.VirtualOther = k.calcVirtualAmountOther(ctx, *ratio.Ratio, denom)
		} else {
			pair.VirtualBase = k.calcVirtualAmountBase(ctx, *ratio.Ratio, denom)
			pair.VirtualOther = math.LegacyZeroDec()
		}

		k.SetLiquidityPair(ctx, pair)

		if ordersCaches != nil {
			ordersCaches.LiquidityPair.Set(denom, pair)
		}
	} else if !found || ratio.Ratio == nil {
		liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		k.initPair(ctx, liqBase, denom)
	}
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

func (k Keeper) PairRatio(ctx context.Context, denom string) *math.LegacyDec {
	liqBase := k.GetFullLiquidityBase(ctx, denom)
	liqOther := k.GetFullLiquidityOther(ctx, denom)
	if !liqOther.GT(math.LegacyZeroDec()) {
		return nil
	}

	ratio := liqBase.Mul(liqOther)
	return &ratio
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
