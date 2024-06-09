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

func (k Keeper) GetRatio(ctx context.Context, denom string) (val types.Ratio, found bool) {
	return k.ratios.Get(ctx, denom)
}

func (k Keeper) GetAllRatio(ctx context.Context) (list []types.Ratio) {
	iterator := k.ratios.Iterator(ctx)
	for iterator.Valid() {
		list = append(list, iterator.GetNext())
	}

	return
}

func compareRatios(r1, r2 types.Ratio) bool {
	if r1.Denom != r2.Denom {
		return false
	}

	if r1.Ratio != nil && r2.Ratio != nil {
		if !r1.Ratio.Equal(*r2.Ratio) {
			return false
		}
	}

	if r1.Ratio != nil && r2.Ratio == nil {
		return false
	}

	if r1.Ratio == nil && r2.Ratio != nil {
		return false
	}

	return true
}
