package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SetLiquidityNextIndex(ctx context.Context, nextindex types.LiquidityNextIndex) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityNextIndex))
	b := k.cdc.MustMarshal(&nextindex)
	store.Set([]byte{0}, b)
}

func (k Keeper) GetLiquidityNextIndex(ctx context.Context) (val types.LiquidityNextIndex, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityNextIndex))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}
