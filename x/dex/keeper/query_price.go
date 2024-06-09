package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Price(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	price, err := k.CalculatePrice(ctx, req.DenomFrom, req.DenomTo)
	if err != nil {
		return nil, err
	}

	return &types.QueryPriceResponse{Price: price.String()}, nil
}

func (k Keeper) PriceUsd(goCtx context.Context, req *types.QueryPriceUsdRequest) (*types.QueryPriceUsdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	response := types.QueryPriceUsdResponse{}

	ctx := sdk.UnwrapSDKContext(goCtx)
	price, err := k.GetPriceInUSD(ctx, req.Denom)
	if err != nil {
		if !errors.Is(err, types.ErrNilRatio) {
			return nil, err
		}
	} else {
		response.Price = price.String()
	}

	return &response, nil
}
