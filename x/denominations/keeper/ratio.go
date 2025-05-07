package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k Keeper) SetRatio(ctx context.Context, ratio types.Ratio) {
	k.ratios.Set(ctx, ratio.Denom, ratio)
}

func (k Keeper) RemoveRatio(ctx context.Context, denom string) {
	k.ratios.Remove(ctx, denom)
}

func (k Keeper) GetRatio(ctx context.Context, denom string) (types.Ratio, error) {
	ratio, has := k.ratios.Get(ctx, denom)
	if !has {
		return types.Ratio{}, fmt.Errorf("no ratio found: %v", denom)
	}

	return ratio, nil
}

func (k Keeper) GetAllRatios(ctx context.Context) (list []types.Ratio) {
	return k.ratios.Iterator(ctx, nil).GetAll()
}
