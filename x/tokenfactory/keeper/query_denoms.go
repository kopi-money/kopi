package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Denoms(goCtx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	iterator := k.DenomIterator(ctx)
	defer iterator.Close()

	store := k.DenomStore(ctx)
	response := types.QueryDenomsResponse{}

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var denom types.FactoryDenom
		if err := k.cdc.Unmarshal(value, &denom); err != nil {
			return err
		}

		response.Denoms = append(response.Denoms, &denom)

		return nil
	})

	if err != nil {
		return nil, err
	}

	response.Pagination = pageRes
	return &response, nil
}
