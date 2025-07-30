package keeper

import (
	"context"

	"github.com/kopi-money/constants"

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

func (k Keeper) getTradeFeeShareStakers(ctx context.Context, denom string) math.LegacyDec {
	if denom == constants.BaseCurrency {
		return k.getTradeFeeShareBase(ctx)
	} else {
		return k.getTradeFeeShareOther(ctx)
	}
}

func (k Keeper) getTradeFeeShareBase(ctx context.Context) math.LegacyDec {
	share := k.GetParams(ctx).TradeFeeBaseIncomeShareToStakers
	if share.IsNil() || !share.IsPositive() {
		return types.TradeFeeBaseIncomeShareToStakers
	}

	return share
}

func (k Keeper) getTradeFeeShareOther(ctx context.Context) math.LegacyDec {
	share := k.GetParams(ctx).TradeFeeOtherIncomeShareToStakers
	if share.IsNil() || !share.IsPositive() {
		return types.TradeFeeOtherIncomeShareToStakers
	}

	return share
}

func (k Keeper) getSellAmount(ctx context.Context, denom string) (math.Int, bool) {
	for _, rsa := range k.GetParams(ctx).ReserveSellAmounts {
		if rsa.Denom == denom {
			return rsa.SellAmount, true
		}
	}

	return math.Int{}, false
}
