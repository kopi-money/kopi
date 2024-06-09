package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) OrdersAddress(goCtx context.Context, req *types.QueryOrdersAddressRequest) (*types.QueryOrdersAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	orders := []*types.OrderResponse{}
	for _, order := range k.GetAllOrdersByAddress(ctx, req.Address) {
		if order.Creator == req.Address {
			orderResponse, err := k.toOrderResponse(ctx, order)
			if err != nil {
				return nil, err
			}

			orders = append(orders, orderResponse)
		}
	}

	return &types.QueryOrdersAddressResponse{Orders: orders}, nil
}
