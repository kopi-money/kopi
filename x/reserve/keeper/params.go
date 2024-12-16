package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/reserve/types"
)

func (k Keeper) GetParams(ctx context.Context) types.Params {
	params, has := k.params.Get(ctx)
	if !has {
		return types.DefaultParams()
	}

	return params
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.params.Set(ctx, params)
	return nil
}

func (k Keeper) getKCoinBurnShare(ctx context.Context) math.LegacyDec {
	kCoinBurnShare := k.GetParams(ctx).KcoinBurnShare
	if kCoinBurnShare.IsNil() {
		kCoinBurnShare = math.LegacyOneDec()
	}

	return kCoinBurnShare
}

func (k Keeper) sellThreshold(ctx context.Context) math.LegacyDec {
	sellThreshold := k.GetParams(ctx).SellThreshold
	if sellThreshold.IsNil() {
		sellThreshold = math.LegacyOneDec()
	}

	return sellThreshold
}

func (k Keeper) buyThreshold(ctx context.Context) math.LegacyDec {
	buyThreshold := k.GetParams(ctx).BuyThreshold
	if buyThreshold.IsNil() {
		buyThreshold = math.LegacyNewDecWithPrec(9999, 4) // 0.9999
	}

	return buyThreshold
}
