package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) NewOrdersCaches(ctx context.Context) *types.OrdersCaches {
	return types.NewOrderCaches(
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
		func(denom string, _ ...any) math.Int {
			acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
			return k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(denom)
		},
		func(denom string) []types.Liquidity {
			return k.liquidityEntries.Iterator(ctx, nil, denom).GetAll()
		},
		func(denom string, _ ...any) math.LegacyDec {
			return k.getMovingLiquidity(ctx, denom).Amount
		},
		func() []types.DiscountLevel {
			return k.GetParams(ctx).DiscountLevels
		},
		func(denom string, _ ...any) math.LegacyDec {
			return k.DenomKeeper.MinTradeLiquidity(ctx, denom).ToLegacyDec()
		},
	)
}
