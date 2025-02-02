package keeper

import (
	"context"
	"sort"

	"github.com/kopi-money/kopi/x/denominations/types"
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

	sort.SliceStable(params.DexDenoms, func(i, j int) bool {
		return params.DexDenoms[i].Name < params.DexDenoms[j].Name
	})

	sort.SliceStable(params.CollateralDenoms, func(i, j int) bool {
		return params.CollateralDenoms[i].DexDenom < params.CollateralDenoms[j].DexDenom
	})

	k.params.Set(ctx, params)
	return nil
}
