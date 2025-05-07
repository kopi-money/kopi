package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

// UpdateMovingLiquidities updates the moving liquidity values for each denom. When, for exapple, there 1000 kUSD
// liquidity and another 1000 is added, then the newly added kUSD won't be used for price calculation right away to
// prevent certain scenarios of price manipulation.
// The moving liquidity is updated as follows (using an update factor of 0.99 as example):
// - Current provided liquidity: 2000 kUSD
// - Current moving liquidity: 1000 kUSD
// - New moving liquidity: 2000 * 0.01 + 1000 * 0.99 = 1010
func (k Keeper) UpdateMovingLiquidities(ctx context.Context) {
	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	poolBalances := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	decayDeposits := k.getLiquiditySpreadDecayFromDeposits(ctx)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		balance := poolBalances.AmountOf(denom)
		movingLiquidity := k.getMovingLiquidity(ctx, denom)
		movingLiquidity.DepositAmount = updateLiquidityFromDeposits(movingLiquidity.DepositAmount, balance.ToLegacyDec(), decayDeposits)
		k.movingLiquidity.Set(ctx, denom, movingLiquidity)
	}
}

func (k Keeper) getMovingLiquidity(ctx context.Context, denom string) types.MovingLiquidity {
	movingLiquidity, has := k.movingLiquidity.Get(ctx, denom)
	if !has {
		return types.MovingLiquidity{
			DepositAmount: math.LegacyZeroDec(),
		}
	}

	return movingLiquidity
}

func updateLiquidityFromDeposits(moving, balance, decay math.LegacyDec) math.LegacyDec {
	if !moving.IsPositive() || balance.LTE(moving) {
		return balance
	}

	ml1 := moving.Mul(decay)
	ml2 := balance.Mul(math.LegacyOneDec().Sub(decay))
	return ml1.Add(ml2)
}
