package keeper

import (
	"context"
	"github.com/pkg/errors"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) OrdersSum(goCtx context.Context, req *types.QueryOrdersSumRequest) (*types.QueryOrdersSumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	sum := math.LegacyZeroDec()
	for _, order := range k.GetAllOrders(ctx) {
		value, err := k.GetValueInUSD(ctx, order.DenomFrom, order.AmountLeft)
		if err != nil {
			return nil, errors.Wrap(err, "culd not get order value in usd")
		}

		sum = sum.Add(value)
	}

	return &types.QueryOrdersSumResponse{Sum: sum.String()}, nil
}

func (k Keeper) OrdersDenomSum(goCtx context.Context, req *types.QueryOrdersDenomSumRequest) (*types.QueryOrdersDenomSumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ordersMap := make(map[string]math.Int)

	for _, order := range k.GetAllOrders(ctx) {
		ordersMap[order.DenomFrom] = ordersMap[order.DenomFrom].Add(order.AmountLeft)
	}

	orderSums := []*types.OrdersSum{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		sum := "0"
		orderSum, has := ordersMap[denom]
		if has {
			sum = orderSum.String()
		}

		orderSums = append(orderSums, &types.OrdersSum{
			DenomFrom: denom,
			Sum:       sum,
		})
	}

	return &types.QueryOrdersDenomSumResponse{Denoms: orderSums}, nil
}
