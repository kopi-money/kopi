package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Ratio(ctx context.Context, req *types.QueryGetRatioRequest) (*types.QueryGetRatioResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetRatio(ctx, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetRatioResponse{Ratio: types.RatioResponse{
		Denom: req.Denom,
		Ratio: val.String(),
	}}, nil
}

func (k Keeper) Ratios(ctx context.Context, _ *types.QueryGetRatiosRequest) (*types.QueryGetRatiosResponse, error) {
	var ratios []types.RatioResponse

	for _, ratio := range k.GetAllRatio(ctx) {
		ratios = append(ratios, types.RatioResponse{
			Denom: ratio.Denom,
			Ratio: ratio.Ratio.String(),
		})
	}

	return &types.QueryGetRatiosResponse{Ratios: ratios}, nil
}
