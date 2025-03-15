package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Price(ctx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	price, err := k.DenomKeeper.CalculatePrice(ctx, req.DenomGiving, req.DenomReceiving)
	if err != nil {
		return nil, err
	}

	return &types.QueryPriceResponse{
		Price: price.String(),
	}, nil
}

func (k Keeper) PriceUsd(ctx context.Context, req *types.QueryPriceUsdRequest) (*types.QueryPriceUsdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	price, err := k.DenomKeeper.GetPriceInUSD(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryPriceUsdResponse{
		Price: price.String(),
	}, nil
}
