package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) OrderPool(ctx context.Context, _ *types.QueryOrderPoolRequest) (*types.QueryOrderPoolResponse, error) {
	orderCoins := k.OrderSum(ctx)

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	coins := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())

	balances := []types.OrderBalance{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		balance := types.OrderBalance{}
		balance.Denom = denom

		has, coin := coins.Find(denom)
		if has {
			balance.PoolAmount = coin.Amount.String()
		} else {
			balance.PoolAmount = math.ZeroInt().String()
		}

		if orderSum, exists := orderCoins[denom]; exists {
			balance.SumOrder = orderSum.String()
		} else {
			balance.SumOrder = math.LegacyZeroDec().String()
		}

		balances = append(balances, balance)
	}

	return &types.QueryOrderPoolResponse{
		Balance: balances,
	}, nil
}

func (k Keeper) OrderSum(ctx context.Context) map[string]math.Int {
	coins := make(map[string]math.Int)

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		if _, exists := coins[order.DenomGiving]; !exists {
			coins[order.DenomGiving] = math.ZeroInt()
		}

		coins[order.DenomGiving] = coins[order.DenomGiving].Add(order.AmountLeft)
	}

	return coins
}
