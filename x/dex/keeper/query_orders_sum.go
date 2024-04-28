package keeper

import (
	"context"
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

	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	sum := math.LegacyZeroDec()
	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &order)

		price, _ := k.GetPriceInUSD(ctx, order.DenomFrom)

		if order.AmountLeft.GT(math.ZeroInt()) {
			sum = sum.Add(price.Quo(order.AmountLeft.ToLegacyDec()))
		}
	}

	return &types.QueryOrdersSumResponse{Sum: sum.String()}, nil
}

func (k Keeper) OrdersDenomSum(goCtx context.Context, req *types.QueryOrdersDenomSumRequest) (*types.QueryOrdersDenomSumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ordersMap := make(map[string]math.Int)

	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &order)

		ordersMap[order.DenomFrom] = ordersMap[order.DenomFrom].Add(order.AmountLeft)
	}

	orderSums := []*types.OrdersSum{}
	for denom, sum := range ordersMap {
		orderSums = append(orderSums, &types.OrdersSum{
			DenomFrom: denom,
			Sum:       sum.String(),
		})
	}

	return &types.QueryOrdersDenomSumResponse{Denoms: orderSums}, nil
}
