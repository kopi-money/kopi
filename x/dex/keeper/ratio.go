package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SetRatio(ctx context.Context, ratio types.Ratio) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRatio))
	b := k.cdc.MustMarshal(&ratio)
	store.Set(types.KeyString(ratio.Denom), b)
}

func (k Keeper) RemoveRatio(ctx context.Context, ratio types.Ratio) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRatio))
	store.Delete(types.KeyString(ratio.Denom))
}

func (k Keeper) GetRatio(ctx context.Context, denom string) (val types.Ratio, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRatio))

	b := store.Get(types.KeyString(denom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) GetAllRatio(ctx context.Context) (list []types.Ratio) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRatio))

	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Ratio
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
