package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Orders(ctx context.Context, req *types.QueryOrdersRequest) (*types.QueryOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var orders []*types.OrderResponse

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		orderResponse, err := k.toOrderResponse(ctx, order)
		if err != nil {
			return nil, err
		}

		orders = append(orders, orderResponse)
	}

	return &types.QueryOrdersResponse{Orders: orders}, nil
}

func (k Keeper) OrdersNum(_ context.Context, req *types.QueryOrdersNumRequest) (*types.QueryOrdersNumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryOrdersNumResponse{Num: int64(k.GetAllOrdersNum())}, nil
}

func (k Keeper) toOrderResponse(ctx context.Context, order types.Order) (*types.OrderResponse, error) {
	amountLeftUSD, err := k.GetValueInUSD(ctx, order.DenomFrom, order.AmountLeft.ToLegacyDec())
	if err != nil {
		return nil, errors.Wrap(err, "could not get amount received in usd")
	}

	return &types.OrderResponse{
		Index:           order.Index,
		Creator:         order.Creator,
		DenomFrom:       order.DenomFrom,
		DenomTo:         order.DenomTo,
		TradeAmount:     order.TradeAmount.String(),
		AmountLeft:      order.AmountLeft.String(),
		AmountLeftUsd:   amountLeftUSD.String(),
		MaxPrice:        order.MaxPrice.String(),
		NumBlocks:       order.NumBlocks,
		BlockEnd:        uint64(order.AddedAt) + order.NumBlocks,
		AllowIncomplete: order.AllowIncomplete,
	}, nil
}

func (k Keeper) OrdersByPair(goCtx context.Context, req *types.OrdersByPairRequest) (*types.QueryOrdersByPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var asks, bids []*types.OrderResponse

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()

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
