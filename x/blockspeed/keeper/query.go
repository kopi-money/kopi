package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/blockspeed/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Blockspeed(ctx context.Context, _ *types.QueryBlockspeedRequest) (*types.QueryBlockspeedResponse, error) {
	blockspeed := k.GetBlockspeed(ctx)

	secondsPerBlock := blockspeed.AverageTime.Quo(math.LegacyNewDec(1000)) // C
	blocksPerSecond := math.LegacyOneDec().Quo(secondsPerBlock)            // C

	return &types.QueryBlockspeedResponse{
		BlocksPerSecond: blocksPerSecond.String(),
		SecondsPerBlock: secondsPerBlock.String(),
	}, nil
}

func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
