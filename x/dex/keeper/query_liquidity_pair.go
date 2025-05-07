package keeper

import (
	"context"
	"github.com/kopi-money/kopi/constants"

	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityPairAll(ctx context.Context, _ *types.QueryAllLiquidityPairRequest) (*types.QueryAllLiquidityPairResponse, error) {
	return &types.QueryAllLiquidityPairResponse{
		LiquidityPair: k.GetAllLiquidityPair(ctx),
	}, nil
}

func (k Keeper) LiquidityPair(ctx context.Context, req *types.QueryGetLiquidityPairRequest) (*types.QueryGetLiquidityPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	liquidityPair, err := k.GetLiquidityPair(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	fullOther := k.GetPoolLiquidity(ctx, req.Denom)
	fullBase := k.GetPoolLiquidity(ctx, constants.BaseCurrency)

	return &types.QueryGetLiquidityPairResponse{
		VirtualBase:  liquidityPair.Base.Virtual.String(),
		VirtualOther: liquidityPair.Other.Virtual.String(),
		FullBase:     fullBase.String(),
		FullOther:    fullOther.String(),
	}, nil
}
