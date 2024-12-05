package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/ls/types"
)

func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{
		Params: k.GetParams(ctx),
	}, nil
}
