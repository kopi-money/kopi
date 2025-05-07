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
		movingLiquidity.Amount = updateLiquidityFromDeposits(movingLiquidity.Amount, balance.ToLegacyDec(), decayDeposits)
		k.movingLiquidity.Set(ctx, denom, movingLiquidity)
	}
}

func (k Keeper) updateMovingLiquidityFromTrade(ctx context.Context, denom string, amount math.LegacyDec) {
	movingLiquidity := k.getMovingLiquidity(ctx, denom)
	movingLiquidity.Amount = movingLiquidity.Amount.Add(amount)
	movingLiquidity.Amount = math.LegacyMaxDec(movingLiquidity.Amount, math.LegacyZeroDec())
	k.movingLiquidity.Set(ctx, denom, movingLiquidity)
}

func (k Keeper) SetMovingLiquidity(ctx context.Context, denom string, amount math.LegacyDec) {
	k.movingLiquidity.Set(ctx, denom, types.MovingLiquidity{Amount: amount})
}

func (k Keeper) getMovingLiquidity(ctx context.Context, denom string) types.MovingLiquidity {
	movingLiquidity, has := k.movingLiquidity.Get(ctx, denom)
	if !has {
		return types.MovingLiquidity{
			Amount: math.LegacyZeroDec(),
		}
	}

	return movingLiquidity
}

func (k Keeper) capMovingLiquidity(ctx context.Context, denom string) math.LegacyDec {
	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	poolBalance := k.BankKeeper.SpendableCoin(ctx, poolAcc.GetAddress(), denom).Amount.ToLegacyDec()

	movingLiquidity := k.getMovingLiquidity(ctx, denom)
	movingLiquidity.Amount = math.LegacyMinDec(poolBalance, movingLiquidity.Amount)
	k.movingLiquidity.Set(ctx, denom, movingLiquidity)
	return movingLiquidity.Amount
}

func (k Keeper) exportMovingLiquidity(ctx context.Context) (list []types.GenesisMovingLiquidity) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {

		list = append(list, types.GenesisMovingLiquidity{
			Denom:  denom,
			Amount: k.getMovingLiquidity(ctx, denom).Amount,
		})
	}

	return
}

func updateLiquidityFromDeposits(moving, balance, factor math.LegacyDec) math.LegacyDec {
	f1 := factor
	f2 := math.LegacyOneDec().Sub(factor)

	s1 := f1.Mul(moving)
	s2 := f2.Mul(balance)
	return s1.Add(s2)
}
