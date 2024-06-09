package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetLiquiditySum(ctx context.Context, denom string) math.Int {
	liq, has := k.liquiditySums.Get(ctx, denom)
	if !has {
		return math.ZeroInt()
	}

	return liq.Amount
}

func (k Keeper) SetLiquiditySum(ctx context.Context, liquiditySum types.LiquiditySum) {
	k.liquiditySums.Set(ctx, liquiditySum.Denom, liquiditySum)
}

// SumLiquidity is used in testing
func (k Keeper) SumLiquidity(ctx context.Context, denom string) math.Int {
	liqSum := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		liqSum = liqSum.Add(liq.Amount)
	}

	return liqSum
}

func compareLiquiditySums(ls1, ls2 types.LiquiditySum) bool {
	return ls1.Denom == ls2.Denom && ls1.Amount.Equal(ls2.Amount)
}
