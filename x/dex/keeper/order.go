package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SetOrder sets a specific order in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetOrder(ctx context.Context, order types.Order) (uint64, error) {
	if order.Index == 0 {
		nextIndex, _ := k.ordersNextIndex.Get(ctx)
		nextIndex += 1
		k.ordersNextIndex.Set(ctx, nextIndex)
		order.Index = nextIndex
	}

	k.orders.Set(ctx, order.Index, order)

	if err := k.CheckOrderPoolBalance(ctx, order.DenomFrom); err != nil {
		return order.Index, err
	}

	return order.Index, nil
}

// GetOrder returns a order from its id
func (k Keeper) GetOrder(ctx context.Context, index uint64) (val types.Order, found bool) {
	return k.orders.Get(ctx, index)
}

// RemoveOrder removes a order from the store
func (k Keeper) RemoveOrder(ctx context.Context, order types.Order) {
	k.orders.Remove(ctx, order.Index)
}

func (k Keeper) CheckOrderPoolBalance(ctx context.Context, denom string) error {
	var (
		poolAmount int64
		sumOrder   int64
	)

	iterator := k.orders.Iterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		if order.DenomFrom != denom {
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
		return fmt.Errorf("%v / %v", poolAmount, sumOrder)
	}

	return nil
}

func (k Keeper) ordersSum(ctx context.Context) map[string]math.Int {
	denomSums := make(map[string]math.Int)

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		sum, has := denomSums[order.DenomFrom]
		if !has {
			sum = math.ZeroInt()
		}

		denomSums[order.DenomFrom] = sum.Add(order.AmountLeft)
	}

	return denomSums
}

func (k Keeper) OrderIterator(ctx context.Context) *cache.Iterator[uint64, types.Order] {
	return k.orders.Iterator(ctx)
}

func (k Keeper) GetAllOrdersNum() int {
	return k.orders.Size()
}

func (k Keeper) GetAllOrdersByAddress(ctx context.Context, address string) (list []types.Order) {
	iterator := k.orders.Iterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		if order.Creator == address {
			list = append(list, order)
		}
	}

	return
}

func compareOrders(o1, o2 types.Order) bool {
	return o1.AddedAt == o2.AddedAt &&
		o1.AllowIncomplete == o2.AllowIncomplete &&
		o1.AmountLeft.Equal(o2.AmountLeft) &&
		o1.Creator == o2.Creator &&
		o1.DenomFrom == o2.DenomFrom &&
		o1.DenomTo == o2.DenomTo &&
		o1.ExecutionInterval == o2.ExecutionInterval &&
		o1.Index == o2.Index &&
		o1.MaxPrice.Equal(o2.MaxPrice) &&
		o1.NumBlocks == o2.NumBlocks &&
		o1.TradeAmount.Equal(o2.TradeAmount)
}
