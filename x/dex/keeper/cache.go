package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) NewOrdersCaches(ctx context.Context) *types.OrdersCaches {
	return types.NewOrderCaches(
		func() sdk.Coins {
			dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
			return k.BankKeeper.SpendableCoins(ctx, dexAcc.GetAddress())
		},
		func(denom string) types.LiquidityPair {
			liq, _ := k.GetLiquidityPair(ctx, denom)
			return liq
		},
		func(other string) math.LegacyDec {
			return k.GetFullLiquidityBase(ctx, other)
		},
		func(denom string) math.LegacyDec {
			return k.GetFullLiquidityOther(ctx, denom)
		},
		func(denom string) []types.Liquidity {
			return k.LiquidityIterator(ctx, denom).GetAll()
		},
	)
}
