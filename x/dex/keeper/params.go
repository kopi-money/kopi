package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/kopi-money/kopi/x/dex/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

func (k Keeper) GetTradeFee(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).TradeFee
}

func (k Keeper) GetReserveFeeShare(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).ReserveShare
}

func (k Keeper) getProviderFee(ctx context.Context) math.LegacyDec {
	return k.GetTradeFee(ctx).Mul(k.GetReserveFeeShare(ctx))
}
