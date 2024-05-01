package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetTotalValueLocked(goCtx context.Context, req *types.GetTotalValueLockedQuery) (*types.GetTotalValueLockedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	totalDeposited, err := k.getTotalDeposited(ctx)
	if err != nil {
		return nil, err
	}

	totalCollateral, err := k.getTotalCollateral(ctx)
	if err != nil {
		return nil, err
	}

	totalValueLocked := totalDeposited.Add(totalCollateral)
	return &types.GetTotalValueLockedResponse{Sum: totalValueLocked.String()}, nil
}

func (k Keeper) getTotalDeposited(ctx sdk.Context) (math.LegacyDec, error) {
	total := math.LegacyZeroDec()

	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		available := k.GetVaultAmount(ctx, CAsset)
		availableUSD, err := k.DexKeeper.GetValueInUSD(ctx, CAsset.BaseDenom, available)
		if err != nil {
			return total, err
		}

		borrowed := k.GetLoansSum(ctx, CAsset.BaseDenom)
		borrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, CAsset.BaseDenom, borrowed.RoundInt())
		if err != nil {
			return total, err
		}

		total = total.Add(availableUSD)
		total = total.Add(borrowedUSD)
	}

	return total, nil
}

func (k Keeper) getTotalCollateral(ctx sdk.Context) (math.LegacyDec, error) {
	total := math.LegacyZeroDec()

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		sum := k.getCollateralSum(ctx, denom.Denom)
		sumUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, sum)
		if err != nil {
			return total, err
		}

		total = total.Add(sumUSD)
	}

	return total, nil
}
