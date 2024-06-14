package keeper

import (
	"context"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SetRatio(ctx context.Context, ratio types.Ratio) {
	k.ratios.Set(ctx, ratio.Denom, ratio)
}

func (k Keeper) RemoveRatio(ctx context.Context, ratio types.Ratio) {
	k.ratios.Remove(ctx, ratio.Denom)
}

func (k Keeper) GetRatio(ctx context.Context, denom string) (types.Ratio, error) {
	ratio, has := k.ratios.Get(ctx, denom)
	if !has {
		fac, err := k.DenomKeeper.InitialVirtualLiquidityFactor(ctx, denom)
		if err != nil {
			return types.Ratio{}, err
		}

		ratio = types.Ratio{
			Denom: denom,
			Ratio: fac,
		}
	}

	return ratio, nil
}

func (k Keeper) GetAllRatio(ctx context.Context) (list []types.Ratio) {
	return k.ratios.Iterator(ctx, nil, nil).GetAll()
}

func compareRatios(r1, r2 types.Ratio) bool {
	if r1.Denom != r2.Denom {
		return false
	}

	return r1.Ratio.Equal(r2.Ratio)
}
