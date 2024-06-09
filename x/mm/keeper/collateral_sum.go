package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) getCollateralSum(ctx context.Context, denom string) math.Int {
	sum, has := k.collateralSum.Get(ctx, denom)
	if !has {
		return math.ZeroInt()
	}

	return sum.Sum
}

func (k Keeper) setCollateralSum(ctx context.Context, denom string, amount math.Int) {
	k.collateralSum.Set(ctx, denom, types.CollateralSum{Sum: amount})
}

func compareCollateralSums(cs1, cs2 types.CollateralSum) bool {
	return cs1.Sum.Equal(cs2.Sum)
}
