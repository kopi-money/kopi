package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCreditLineUsage(ctx context.Context, req *types.GetCreditLineUsageQuery) (*types.GetCreditLineUsageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	userLoanSum, _, err := k.getUserLoansSumUSD(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrap(err, "could not get user loan sum")
	}

	_, collateralUserSum, err := k.getCollateralUserSumUSD(ctx, req.Address)
	if err != nil {
		return nil, errors.Wrap(err, "could not get user loan sum")
	}

	creditLineUsage := math.LegacyZeroDec()
	if collateralUserSum.GT(math.LegacyZeroDec()) {
		creditLineUsage = userLoanSum.Quo(collateralUserSum)
	}

	return &types.GetCreditLineUsageResponse{Usage: creditLineUsage.String()}, nil
}
