package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) getCollateralSum(ctx context.Context, denom string) math.Int {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral)
	return k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(denom)
}
