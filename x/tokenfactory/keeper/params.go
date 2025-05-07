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

func (k Keeper) getMinimumPoolSize(ctx context.Context) math.Int {
	minimumPoolSize := k.GetParams(ctx).MinimumPoolSize
	if minimumPoolSize.IsNil() {
		return types.MinimumPoolSize
	}

	return minimumPoolSize
}

func (k Keeper) getMinimumPoolFee(ctx context.Context) math.LegacyDec {
	minimumPoolFee := k.GetParams(ctx).MinimumPoolFee
	if minimumPoolFee.IsNil() {
		return types.MinimumPoolFee
	}

	return minimumPoolFee
}

func (k Keeper) getMaximumPoolFee(ctx context.Context) math.LegacyDec {
	maximumPoolFee := k.GetParams(ctx).MaximumPoolFee
	if maximumPoolFee.IsNil() {
		return types.MaximumPoolFee
	}

	return maximumPoolFee
}

func (k Keeper) getMinimumPoolMovingValue(ctx context.Context) math.Int {
	minimumPoolMovingValue := k.GetParams(ctx).MinimumPoolMovingValue
	if minimumPoolMovingValue.IsNil() {
		return types.MinimumPoolMovingValue
	}

	return minimumPoolMovingValue
}

func (k Keeper) getReserveFeeShare(ctx context.Context) math.LegacyDec {
	reserveFeeShare := k.GetParams(ctx).ReserveFeeShare
	if reserveFeeShare.IsNil() || !reserveFeeShare.IsPositive() {
		return types.ReserveFeeShare
	}

	return reserveFeeShare
}

func (k Keeper) getMaximumVestingUnlockSteps(ctx context.Context) int64 {
	maximumVestingUnlockSteps := k.GetParams(ctx).MaximumVestingUnlockSteps
	if maximumVestingUnlockSteps < 1 {
		return types.MaximumVestingUnlockSteps
	}

	return maximumVestingUnlockSteps
}

func (k Keeper) getCategory(ctx context.Context, categoryIndex uint64) (types.Category, bool) {
	for _, category := range k.GetParams(ctx).Categories.Categories {
		if category.Index == categoryIndex {
			return category, true
		}
	}

	return types.Category{}, false
}

func (k Keeper) validCategoryIndex(ctx context.Context, categoryIndex uint64) bool {
	for _, category := range k.GetParams(ctx).Categories.Categories {
		if category.Index == categoryIndex {
			return true
		}
	}

	return false
}
