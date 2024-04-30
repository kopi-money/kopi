package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SetLiquidityNextIndex(ctx context.Context, nextIndex uint64) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityNextIndex))
	b := k.cdc.MustMarshal(&types.LiquidityNextIndex{Next: nextIndex})
	store.Set([]byte{0}, b)
}

func (k Keeper) GetLiquidityNextIndex(ctx context.Context) uint64 {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityNextIndex))

	b := store.Get([]byte{0})
	if b == nil {
		return 1
	}

	var val types.LiquidityNextIndex
	k.cdc.MustUnmarshal(b, &val)
	return val.Next
}
