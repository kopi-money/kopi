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
	liqBase := k.GetSpreadLiquidity(ctx, constants.BaseCurrency)
	liqOther := k.GetSpreadLiquidity(ctx, ratio.Denom)
	extraVirtualLiquidity := k.DenomKeeper.ExtraVirtualLiquidity(ctx, ratio.Denom)

	return k.CreateLiquidityPairWithLiquidity(ctx, ratio, liqBase, liqOther, extraVirtualLiquidity)
}

func (k Keeper) CreateLiquidityPairWithLiquidity(ctx context.Context, ratio denomtypes.Ratio, liqBase, liqOther math.LegacyDec, extraVirtualLiquidity math.Int) (types.LiquidityPair, error) {
	liqOtherInBase := liqOther.Quo(ratio.Ratio)
	liqBase = math.LegacyMinDec(liqBase, liqOtherInBase)
	liqBaseInOther := liqBase.Mul(ratio.Ratio)

	pair := types.LiquidityPair{
		Denom:        ratio.Denom,
		ActualBase:   liqBase,
		ActualOther:  liqOther,
		VirtualBase:  math.LegacyZeroDec(),
		VirtualOther: math.LegacyZeroDec(),
	}

	if liqBase.LT(liqOtherInBase) {
		pair.VirtualBase = liqOtherInBase.Sub(liqBase)
	}

	if liqOther.LT(liqBaseInOther) {
		pair.VirtualOther = liqBaseInOther.Sub(liqOther)
	}

	extraVirtualLiquidityBase, err := k.DenomKeeper.GetValueInFromUSD(ctx, constants.BaseCurrency, extraVirtualLiquidity.ToLegacyDec())
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
