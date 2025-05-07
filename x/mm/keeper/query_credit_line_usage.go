package keeper

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("get user loan sum: %w", err)
	}

	_, collateralUserSum, err := k.getCollateralUserSumUSD(ctx, req.Address)
	if err != nil {
		return nil, fmt.Errorf("get user loan sum: %w", err)
	}

	creditLineUsage := math.LegacyZeroDec()
	if collateralUserSum.IsPositive() {
		creditLineUsage = userLoanSum.Quo(collateralUserSum) // C
	}

	return &types.GetCreditLineUsageResponse{
		Usage: creditLineUsage.String(),
	}, nil
}

func (k Keeper) CalculateCreditLineUsage(ctx context.Context, address string) (math.LegacyDec, error) {
	_, collateralUserSum, err := k.getCollateralUserSumUSD(ctx, address)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("get user loan sum: %w", err)
	}

	if !collateralUserSum.IsPositive() {
		return math.LegacyZeroDec(), nil
	}

	userLoanSum, _, err := k.getUserLoansSumUSD(ctx, address)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("get user loan sum: %w", err)
	}

	return userLoanSum.Quo(collateralUserSum), nil // C
}
