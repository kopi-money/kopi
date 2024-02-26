package keeper

import (
	"context"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetBorrowInterestRate(ctx context.Context, req *types.GetBorrowInterestRateQuery) (*types.GetBorrowInterestRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	utilityRate := k.calculateUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)

	return &types.GetBorrowInterestRateResponse{InterestRate: interestRate.String()}, nil
}
