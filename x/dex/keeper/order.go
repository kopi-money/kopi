package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetOrderNextIndex(ctx context.Context) uint64 {
	nextIndex, _ := k.ordersNextIndex.Get(ctx)
	return nextIndex
}

func (k Keeper) SetOrderNextIndex(ctx context.Context, index uint64) {
	k.ordersNextIndex.Set(ctx, index)
}

// SetOrder sets a specific order in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetOrder(ctx context.Context, order types.Order) uint64 {
	if order.Index == 0 {
		nextIndex, _ := k.ordersNextIndex.Get(ctx)
		nextIndex += 1
		k.ordersNextIndex.Set(ctx, nextIndex)
		order.Index = nextIndex
	}

	k.orders.Set(ctx, order.Index, order)
	return order.Index
}

// GetOrder returns a order from its id
func (k Keeper) GetOrder(ctx context.Context, index uint64) (val types.Order, found bool) {
	return k.orders.Get(ctx, index)
}

// RemoveOrder removes a order from the store
func (k Keeper) RemoveOrder(ctx context.Context, order types.Order) error {
	if !order.AmountLocked.IsNil() && order.AmountLocked.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(order.DenomGiving, order.AmountLocked))
		address, _ := sdk.AccAddressFromBech32(order.Creator)
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOrders, address, coins); err != nil {
			return err
		}
	}

	k.orders.Remove(ctx, order.Index)
	return nil
}

func (k Keeper) CheckOrderPoolBalance(ctx context.Context, denom string) error {
	var (
		poolAmount int64
		sumOrder   int64
	)

	iterator := k.orders.Iterator(ctx, nil)
	for iterator.Valid() {
		order := iterator.GetNext()
		if order.DenomGiving != denom {
			continue
		}

		sumOrder += order.AmountLeft.Int64()
	}

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	coins := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())

	has, coin := coins.Find(denom)
	if has {
		poolAmount = coin.Amount.Int64()
	}

	diff := poolAmount - sumOrder
	if diff != 0 {
		return fmt.Errorf("denom: %v, poolAmount: %v, %v", denom, poolAmount, sumOrder)
	}

	return nil
}

func (k Keeper) GetOrdersSum(ctx context.Context) map[string]math.Int {
	denomSums := make(map[string]math.Int)

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		sum, has := denomSums[order.DenomGiving]
		if !has {
			sum = math.ZeroInt()
		}

		denomSums[order.DenomGiving] = sum.Add(order.AmountLeft)
	}

	return denomSums
}

func (k Keeper) OrderIterator(ctx context.Context) cache.Iterator[uint64, types.Order] {
	return k.orders.Iterator(ctx, nil)
}

func (k Keeper) OrderCollectionIterator(ctx context.Context, rng collections.Ranger[uint64]) (collections.Iterator[uint64, types.Order], error) {
	return k.orders.CollectionIterator(ctx, rng)
}

func (k Keeper) GetAllOrdersNum() int {
	return k.orders.Size()
}

func (k Keeper) NumRunningOrderTransactions() int {
	return k.orders.NumRunningTransactions()
}

func (k Keeper) GetAllOrdersByAddress(ctx context.Context, address string) (list []types.Order) {
	iterator := k.orders.Iterator(ctx, nil)
	for iterator.Valid() {
		order := iterator.GetNext()
		if order.Creator == address {
			list = append(list, order)
		}
	}

	return
}
