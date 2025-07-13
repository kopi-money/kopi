package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryVestings(ctx context.Context, req *types.QueryVestingsRequest) (*types.QueryVestingsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var vestings []types.Vesting
	iterator := k.vestings.Iterator(ctx, nil)
	for iterator.Valid() {
		vesting := iterator.GetNext()

		if req.FullName != "" && vesting.FactoryDenom != req.FullName {
			continue
		}

		if req.Address != "" && vesting.Address != req.Address {
			continue
		}

		vestings = append(vestings, vesting)
	}

	return &types.QueryVestingsResponse{Vestings: vestings}, nil
}
