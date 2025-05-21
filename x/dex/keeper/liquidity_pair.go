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

	return k.CreateLiquidityPair(ctx, ratio)
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		pair, _ := k.GetLiquidityPair(ctx, denom)
		list = append(list, pair)
	}

	return
}

func (k Keeper) CreateLiquidityPair(ctx context.Context, ratio denomtypes.Ratio) (types.LiquidityPair, error) {
	liqBase := k.GetEffectiveLiquidity(ctx, constants.BaseCurrency)
	liqOther := k.GetEffectiveLiquidity(ctx, ratio.Denom)
	minLiqBase := k.DenomKeeper.MinTradeLiquidity(ctx, constants.BaseCurrency)
	minLiqOther := k.DenomKeeper.MinTradeLiquidity(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, minLiqBase, minLiqOther)
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ctx context.Context, ratio denomtypes.Ratio, liqBase, liqOther math.LegacyDec, minLiqBase, minLiqOther math.Int) (types.LiquidityPair, error) {
	liqOtherInBase := liqOther.Quo(ratio.Ratio)
	liqBase = math.LegacyMinDec(liqBase, liqOtherInBase)
	liqBaseInOther := liqBase.Mul(ratio.Ratio)

	tlBase := types.TradeLiquidity{}
	tlBase.Actual = liqBase
	tlBase.Virtual = math.LegacyZeroDec()

	tlOther := types.TradeLiquidity{}
	tlOther.Actual = liqOther
	tlOther.Virtual = math.LegacyZeroDec()

	if liqBase.LT(liqOtherInBase) {
		tlBase.Virtual = liqOtherInBase.Sub(liqBase)
	}

	if liqOther.LT(liqBaseInOther) {
		tlOther.Virtual = liqBaseInOther.Sub(liqOther)
	}

	//extraVirtualLiquidityBase, err := k.DenomKeeper.GetValueInFromUSD(ctx, constants.BaseCurrency, extraVirtualLiquidity.ToLegacyDec())
	//if err != nil {
	//	return types.LiquidityPair{}, fmt.Errorf("convert extra liq to base: %w", err)
	//}

	//extraVirtualLiquidityOther, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, ratio.Denom, extraVirtualLiquidityBase)
	//if err != nil {
	//	return types.LiquidityPair{}, fmt.Errorf("convert extra liq to base: %w", err)
	//}

	//tlBase.Virtual = tlBase.Virtual.Add(extraVirtualLiquidityBase)
	//tlOther.Virtual = tlOther.Virtual.Add(extraVirtualLiquidityOther)

	if tlBase.GetFull().LT(minLiqBase.ToLegacyDec()) {
		missingBase := minLiqBase.ToLegacyDec().Sub(tlBase.GetFull())
		tlBase.Virtual = tlBase.Virtual.Add(missingBase)

		missingOther, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, ratio.Denom, missingBase)
		if err != nil {
			return types.LiquidityPair{}, fmt.Errorf("convert missing liq to other: %w", err)
		}
		tlOther.Virtual = tlOther.Virtual.Add(missingOther)
	}

	if tlOther.GetFull().LT(minLiqOther.ToLegacyDec()) {
		missingOther := minLiqOther.ToLegacyDec().Sub(tlOther.GetFull())
		tlOther.Virtual = tlOther.Virtual.Add(missingOther)

		missingBase, err := k.DenomKeeper.GetValueIn(ctx, ratio.Denom, constants.BaseCurrency, missingOther)
		if err != nil {
			return types.LiquidityPair{}, fmt.Errorf("convert missing liq to base: %w", err)
		}
		tlBase.Virtual = tlBase.Virtual.Add(missingBase)
	}

	return types.LiquidityPair{
		Base:  tlBase,
		Other: tlOther,
	}, nil
}
