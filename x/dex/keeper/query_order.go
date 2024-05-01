package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Order(goCtx context.Context, req *types.QueryOrderRequest) (*types.QueryOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	order, ok := k.GetOrder(ctx, req.Index)
	if !ok {
		return nil, types.ErrOrderNotFound
	}

	orderResponse, err := k.toOrderResponse(ctx, order)
	if err != nil {
		return nil, err
	}

	return &types.QueryOrderResponse{Order: orderResponse}, nil
}
