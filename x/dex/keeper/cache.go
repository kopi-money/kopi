package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) NewOrdersCaches(ctx context.Context) *types.OrdersCaches {
	oc := types.NewOrderCaches(
		func() sdk.AccAddress {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolTrade)
			return acc.GetAddress()
		},
		func() sdk.AccAddress {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve)
			return acc.GetAddress()
		},
		func() sdk.AccAddress {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
			return acc.GetAddress()
		},
		func() sdk.AccAddress {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
			return acc.GetAddress()
		},
		func() sdk.AccAddress {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeIncome)
			return acc.GetAddress()
		},
		func() math.LegacyDec {
			return k.GetParams(ctx).TradeFee
		},
		func() math.LegacyDec {
			return k.GetParams(ctx).ReserveShare
		},
		func() math.LegacyDec {
			return k.GetParams(ctx).OrderFee
		},
		func() math.LegacyDec {
			return k.getProviderFee(ctx)
		},
		func() *types.CoinMap {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
			coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
			return types.NewCoinMap(coins)
		},
		func(denom string) []types.Liquidity {
			return k.liquidityEntries.Iterator(ctx, nil, denom).GetAll()
		},
	)

	return oc
}
