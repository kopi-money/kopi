package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

const (
	epochTimeMinimum = 60
	epochTimeMaximum = 60 * 60 * 24
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

func (k Keeper) GetTradeFee(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).TradeFee
}

func (k Keeper) GetOrderFee(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).OrderFee
}

func (k Keeper) GetJoinedFee(ctx context.Context) math.LegacyDec {
	factor := math.LegacyOneDec()
	factor = factor.Mul(math.LegacyOneDec().Sub(k.GetTradeFee(ctx)))
	factor = factor.Mul(math.LegacyOneDec().Sub(k.GetOrderFee(ctx)))
	factor = math.LegacyOneDec().Sub(factor)

	return factor
}

func (k Keeper) GetReserveFeeShare(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).ReserveShare
}

func (k Keeper) getProviderFee(ctx context.Context) math.LegacyDec {
	return k.GetTradeFee(ctx).Mul(k.GetReserveFeeShare(ctx))
}

func (k Keeper) getPriceIncreasingFactor(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).PriceIncreasingFactor
}

func (k Keeper) getEpochLength(ctx context.Context) uint64 {
	length := k.GetParams(ctx).EpochLength
	length = max(length, epochTimeMinimum)
	length = min(length, epochTimeMaximum)
	return length
}

func (k Keeper) getLiquiditySpreadDecayFromDeposits(ctx context.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	if params.LiquidityChangeDecayFromDeposits.IsNil() || params.LiquidityChangeDecayFromDeposits.IsZero() {
		return types.LiquidityChangeDecayFromDeposits
	}

	return params.LiquidityChangeDecayFromDeposits
}

func (k Keeper) getMinimumLiquidityLockInBlocks(ctx context.Context) int64 {
	params := k.GetParams(ctx)
	return params.MinimumLiquidityLockInBlocks
}
