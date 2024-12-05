package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/kopi-money/kopi/x/ls/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Undelegations(ctx context.Context, req *types.QueryUndelegationsRequest) (*types.QueryUndelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	undelegations, pageRes, err := query.CollectionPaginate(
		ctx,
		k.undelegations,
		req.Pagination,
		func(index uint64, undelegation types.Undelegation) (*types.GenesisUndelegation, error) {
			return &types.GenesisUndelegation{
				Index:   index,
				Address: undelegation.Address,
				Amount:  undelegation.Amount,
			}, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not get orders from pagination: %w", err)
	}

	return &types.QueryUndelegationsResponse{
		Entries:    undelegations,
		Pagination: pageRes,
	}, nil
}
