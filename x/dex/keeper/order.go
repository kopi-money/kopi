package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SetOrder sets a specific order in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetOrder(ctx context.Context, order types.Order) uint64 {
	if order.Index == 0 {
		order.Index = k.GetOrderNextIndex(ctx)
		k.SetOrderNextIndex(ctx, order.Index+1)
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixOrder))
	b := k.cdc.MustMarshal(&order)
	store.Set(types.KeyIndex(order.Index), b)
	return order.Index
}

// GetOrder returns a order from its id
func (k Keeper) GetOrder(ctx context.Context, index uint64) (val types.Order, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixOrder))
	b := store.Get(types.KeyIndex(index))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveOrder removes a order from the store
func (k Keeper) RemoveOrder(ctx context.Context, order types.Order) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixOrder))
	store.Delete(types.KeyIndex(order.Index))
}

func (k Keeper) OrdersStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixOrder))
}

func (k Keeper) OrdersIterator(ctx context.Context) storetypes.Iterator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixOrder))
	return storetypes.KVStorePrefixIterator(store, []byte{})
}

func (k Keeper) GetAllOrders(ctx context.Context) (list []types.Order) {
	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllOrdersByAddress(ctx context.Context, address string) (list []types.Order) {
	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if val.Creator == address {
			list = append(list, val)
		}
	}

	return
}
