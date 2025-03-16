package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetLiquidityPair(ctx context.Context, denom string) (types.LiquidityPair, error) {
	ratio, err := k.DenomKeeper.GetRatio(ctx, denom)
	if err != nil {
		return types.LiquidityPair{}, fmt.Errorf("get ratio: %w", err)
	}

	pair, err := k.CreateLiquidityPair(ctx, ratio)
	if err != nil {
		return types.LiquidityPair{}, fmt.Errorf("create pair: %w", err)
	}

	return pair, nil
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		pair, _ := k.GetLiquidityPair(ctx, denom)
		list = append(list, pair)
	}

	return
}

func (k Keeper) CreateLiquidityPair(ctx context.Context, ratio denomtypes.Ratio) (types.LiquidityPair, error) {
	liqBase := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	liqOther := k.GetLiquiditySum(ctx, ratio.Denom)
	extraVirtualLiquidity := k.DenomKeeper.ExtraVirtualLiquidity(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, extraVirtualLiquidity)
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ctx context.Context, ratio denomtypes.Ratio, liqBase, liqOther math.Int, extraVirtualLiquidity math.Int) (types.LiquidityPair, error) {
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	liqOtherInBase := liqOtherDec.Quo(ratio.Ratio)
	liqBaseDec = math.LegacyMinDec(liqBaseDec, liqOtherInBase)
	liqBaseInOther := liqBaseDec.Mul(ratio.Ratio)

	pair := types.LiquidityPair{
		Denom:        ratio.Denom,
		ActualBase:   liqBaseDec,
		ActualOther:  liqOtherDec,
		VirtualBase:  math.LegacyZeroDec(),
		VirtualOther: math.LegacyZeroDec(),
	}

	if liqBaseDec.LT(liqOtherInBase) {
		pair.VirtualBase = liqOtherInBase.Sub(liqBaseDec)
	}

	if liqOtherDec.LT(liqBaseInOther) {
		pair.VirtualOther = liqBaseInOther.Sub(liqOtherDec)
	}

	extraVirtualLiquidityUSD := extraVirtualLiquidity.ToLegacyDec().Quo(ratio.Ratio)
	extraVirtualLiquidityBase, err := k.DenomKeeper.GetValueInFromUSD(ctx, constants.BaseCurrency, extraVirtualLiquidityUSD)
	if err != nil {
		return pair, fmt.Errorf("convert extra liq to base: %w", err)
	}

	extraVirtualLiquidityOther, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, ratio.Denom, extraVirtualLiquidityBase)
	if err != nil {
		return pair, fmt.Errorf("convert extra liq to base: %w", err)
	}

	pair.ExtraBase = extraVirtualLiquidityBase
	pair.ExtraOther = extraVirtualLiquidityOther

	return pair, nil
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
