package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var e = math.LegacyNewDecWithPrec(2718281828, 9)

func (k Keeper) CalculateInterestRateForDenom(ctx context.Context, denom string) math.LegacyDec {
	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, denom)
	if err != nil {
		cAsset, err = k.DenomKeeper.GetCAssetByBaseName(ctx, denom)
		if err != nil {
			return math.LegacyZeroDec()
		}
	}

	return k.CalculateInterestRate(ctx, cAsset)
}

func (k Keeper) CalculateInterestRate(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	utilityRate := k.calculateUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)
	return interestRate
}

func (k Keeper) calculateInterestRate(ctx context.Context, utilityRate math.LegacyDec) math.LegacyDec {
	minimumInterestRate := k.GetParams(ctx).MinInterestRate
	a := k.GetParams(ctx).A
	b := k.GetParams(ctx).B

	power := uint64(utilityRate.Mul(a).RoundInt64())
	return minimumInterestRate.Add(e.Power(power).Quo(b)) // C
}

// calculateUtilityRate return the utility rate of a borrowable asset. It gets the sum of given out loans and the
// currently available funds in the vault. The UR then is loan_sum / (loan_sum + funds_in_vault)
func (k Keeper) calculateUtilityRate(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
	borrowableAmount := k.GetVaultAmount(ctx, cAsset)

	sum := loanSum.Add(borrowableAmount.ToLegacyDec())
	if sum.IsZero() {
		return math.LegacyZeroDec()
	}

	return loanSum.Quo(sum) // C
}

func (k Keeper) ApplyInterest(ctx context.Context) error {
	blocksPerYear, err := k.BlockspeedKeeper.BlocksPerYear(ctx)
	if err != nil {
		return fmt.Errorf("could not get blockspeed: %w", err)
	}

	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		k.applyInterestForCAssetLoans(ctx, CAsset, blocksPerYear)
	}

	return nil
}

func (k Keeper) applyInterestForCAssetLoans(ctx context.Context, cAsset *denomtypes.CAsset, blocksPerYear math.LegacyDec) {
	utilityRate := k.calculateUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)
	interestRate = interestRate.Quo(blocksPerYear) // C
	interestRate = interestRate.Add(math.LegacyOneDec())

	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom)
	loanSum.LoanSum = loanSum.LoanSum.Mul(interestRate)
	k.loansSum.Set(ctx, cAsset.BaseDexDenom, loanSum)
}
