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
	extraVirtualLiquidity := k.DenomKeeper.ExtraVirtualLiquidity(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ratio, liqBase, liqOther, extraVirtualLiquidity)
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ratio denomtypes.Ratio, liqBase, liqOther math.Int, extraVirtualLiquidity math.Int) (pair types.LiquidityPair) {
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	liqBaseInOther := liqBaseDec.Mul(ratio.Ratio)
	liqOtherInBase := liqOtherDec.Quo(ratio.Ratio)

	pair.Denom = ratio.Denom
	pair.VirtualBase = math.LegacyZeroDec()
	pair.VirtualOther = math.LegacyZeroDec()

	if liqBaseDec.LT(liqOtherInBase) {
		pair.VirtualBase = liqOtherInBase.Sub(liqBaseDec)
	}

	if liqOtherDec.LT(liqBaseInOther) {
		pair.VirtualOther = liqBaseInOther.Sub(liqOtherDec)
	}

	minVirtualLiquidityBase := extraVirtualLiquidity.ToLegacyDec().Quo(ratio.Ratio)
	pair.ExtraBase = minVirtualLiquidityBase
	pair.ExtraOther = extraVirtualLiquidity.ToLegacyDec()

	return
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

func sumLiquidity(actual, virtual math.LegacyDec) math.LegacyDec {
	if virtual.IsNil() || virtual.IsZero() {
		return actual
	}

	return actual.Add(virtual)
}
