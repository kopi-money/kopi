package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) types.Params {
	params, _ := k.params.Get(ctx)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.params.Set(ctx, params)
	return nil
}

func (k Keeper) getTradeFee(ctx context.Context, poolFee math.LegacyDec) math.LegacyDec {
	reserveFee := k.GetParams(ctx).ReserveFee
	return reserveFee.Add(poolFee)
}

func (k Keeper) getMinimumPoolSize(ctx context.Context) math.Int {
	minimumPoolSize := k.GetParams(ctx).MinimumPoolSize
	if minimumPoolSize.IsNil() {
		return types.MinimumPoolSize
	}

	return minimumPoolSize
}

func (k Keeper) getCreationFee(ctx context.Context) math.Int {
	creationFee := k.GetParams(ctx).CreationFee
	if creationFee.IsNil() {
		return types.CreationFee
	}

	return creationFee
}
