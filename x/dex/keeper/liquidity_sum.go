package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SumLiquidity is used in testing
func (k Keeper) SumLiquidity(ctx context.Context, denom string) math.Int {
	liqSum := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Amount.IsNil() || liq.Amount.IsZero() {
			continue
		}

		liqSum = liqSum.Add(liq.Amount)
	}

	return liqSum
}

func (k Keeper) GetPoolLiquidity(ctx context.Context, denom string) math.Int {
	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	return k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress()).AmountOf(denom)
}

func (k Keeper) GetSpreadLiquidity(ctx context.Context, denom string) math.LegacyDec {
	poolLiquidity := k.GetPoolLiquidity(ctx, denom).ToLegacyDec()
	movingLiquidity := k.getMovingLiquidity(ctx, denom).DepositAmount
	if movingLiquidity.IsZero() {
		return poolLiquidity
	}

	return math.LegacyMinDec(poolLiquidity, movingLiquidity)
}
