package keeper

import (
	"context"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SetLiquiditySum(ctx context.Context, listexmaple types.LiquiditySum) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquiditySum))
	b := k.cdc.MustMarshal(&listexmaple)
	store.Set(types.KeyString(listexmaple.Denom), b)
}

func (k Keeper) GetLiquiditySum(ctx context.Context, denom string) (math.Int, bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquiditySum))
	b := store.Get(types.KeyString(denom))
	if b == nil {
		return math.ZeroInt(), false
	}

	var val types.LiquiditySum
	k.cdc.MustUnmarshal(b, &val)
	return val.Amount, true
}

func (k Keeper) GetAllLiquiditySum(ctx context.Context) (list []types.LiquiditySum) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquiditySum))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.LiquiditySum
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
