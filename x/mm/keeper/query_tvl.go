package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetTotalValueLocked(ctx context.Context, _ *types.GetTotalValueLockedQuery) (*types.GetTotalValueLockedResponse, error) {
	totalDeposited, err := k.getTotalDeposited(ctx)
	if err != nil {
		return nil, err
	}

	totalCollateral, err := k.getTotalCollateral(ctx)
	if err != nil {
		return nil, err
	}

	totalValueLocked := totalDeposited.Add(totalCollateral)
	return &types.GetTotalValueLockedResponse{
		Sum: totalValueLocked.String(),
	}, nil
}

func (k Keeper) getTotalDeposited(ctx context.Context) (math.LegacyDec, error) {
	total := math.LegacyZeroDec()

	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		available := k.GetVaultAmount(ctx, CAsset)
		availableUSD, err := k.DenomKeeper.GetValueInUSD(ctx, CAsset.BaseDexDenom, available.ToLegacyDec())
		if err != nil {
			return total, err
		}

		borrowed := k.GetLoanSumWithDefault(ctx, CAsset.BaseDexDenom).LoanSum
		borrowedUSD, err := k.DenomKeeper.GetValueInUSD(ctx, CAsset.BaseDexDenom, borrowed)
		if err != nil {
			return total, err
		}

		total = total.Add(availableUSD)
		total = total.Add(borrowedUSD)
	}

	return total, nil
}

func (k Keeper) getTotalCollateral(ctx context.Context) (math.LegacyDec, error) {
	total := math.LegacyZeroDec()

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		sum := k.getCollateralSum(ctx, denom.DexDenom)
		sumUSD, err := k.DenomKeeper.GetValueInUSD(ctx, denom.DexDenom, sum.ToLegacyDec())
		if err != nil {
			return total, err
		}

		total = total.Add(sumUSD)
	}

	return total, nil
}
