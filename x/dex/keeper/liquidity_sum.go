package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

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

func (k Keeper) GetLiquiditySum(ctx context.Context, denom string) math.Int {
	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	return k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress()).AmountOf(denom)
}
