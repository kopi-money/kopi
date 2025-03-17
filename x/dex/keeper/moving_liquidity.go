package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

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
