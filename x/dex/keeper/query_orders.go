package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Orders(goCtx context.Context, req *types.QueryOrdersRequest) (*types.QueryOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := k.OrdersStore(ctx)
	orders := []*types.OrderResponse{}

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var order types.Order
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orderResponse, err := k.toOrderResponse(ctx, order)
		if err != nil {
			return err
		}

		orders = append(orders, orderResponse)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryOrdersResponse{Orders: orders, Pagination: pageRes}, nil
}

func (k Keeper) OrdersNum(goCtx context.Context, req *types.QueryOrdersNumRequest) (*types.QueryOrdersNumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	var counter int64 = 0
	for ; iterator.Valid(); iterator.Next() {
		counter++
	}

	return &types.QueryOrdersNumResponse{Num: counter}, nil
}

func (k Keeper) toOrderResponse(ctx context.Context, order types.Order) (*types.OrderResponse, error) {
	amountReceivedUSD, err := k.GetValueInUSD(ctx, order.DenomTo, order.AmountReceived)
	if err != nil {
		return nil, errors.Wrap(err, "could not get amount received in usd")
	}

	amountLeftUSD, err := k.GetValueInUSD(ctx, order.DenomTo, order.AmountLeft)
	if err != nil {
		return nil, errors.Wrap(err, "could not get amount received in usd")
	}

	return &types.OrderResponse{
		Index:             order.Index,
		Creator:           order.Creator,
		DenomFrom:         order.DenomFrom,
		DenomTo:           order.DenomTo,
		TradeAmount:       order.TradeAmount.String(),
		AmountGiven:       order.AmountGiven.String(),
		AmountLeft:        order.AmountLeft.String(),
		AmountLeftUsd:     amountLeftUSD.String(),
		AmountReceived:    order.AmountReceived.String(),
		AmountReceivedUsd: amountReceivedUSD.String(),
		MaxPrice:          order.MaxPrice.String(),
		NumBlocks:         order.NumBlocks,
		BlockEnd:          order.BlockEnd,
		AllowIncomplete:   order.AllowIncomplete,
	}, nil
}

func (k Keeper) OrdersByPair(goCtx context.Context, req *types.OrdersByPairRequest) (*types.QueryOrdersByPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	var asks, bids []*types.OrderResponse

	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &order)

		if order.DenomFrom == req.DenomFrom && order.DenomTo == req.DenomTo {
			orderResponse, err := k.toOrderResponse(ctx, order)
			if err != nil {
				return nil, err
			}

			bids = append(bids, orderResponse)
		}

		if order.DenomFrom == req.DenomTo && order.DenomTo == req.DenomFrom {
			orderResponse, err := k.toOrderResponse(ctx, order)
			if err != nil {
				return nil, err
			}

			asks = append(asks, orderResponse)
		}
	}

	return &types.QueryOrdersByPairResponse{Bids: bids, Asks: asks}, nil
}
