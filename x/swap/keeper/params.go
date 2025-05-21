package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/swap/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) types.Params {
	params, has := k.params.Get(ctx)
	if !has {
		return types.DefaultParams()
	}

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

func (k Keeper) mintThreshold(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).MintThreshold
}

func (k Keeper) burnThreshold(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).BurnThreshold
}

func (k Keeper) parityFactor(ctx context.Context) math.LegacyDec {
	factor := k.GetParams(ctx).ParityFactor
	if factor.IsNil() || factor.IsZero() {
		return types.ParityFactor
	}

	return factor
}
