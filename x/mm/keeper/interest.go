package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var e = math.LegacyNewDecWithPrec(2718281828, 9)

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
	return minimumInterestRate.Add(e.Power(power).Quo(b))
}

// calculateUtilityRate return the utility rate of a borrowable asset. It gets the sum of given out loans and the
// currently available funds in the vault. The UR then is loan_sum / (loan_sum + funds_in_vault)
func (k Keeper) calculateUtilityRate(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum
	borrowableAmount := k.GetVaultAmount(ctx, cAsset)

	sum := loanSum.Add(borrowableAmount.ToLegacyDec())
	if sum.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	return loanSum.Quo(sum)
}

func (k Keeper) ApplyInterest(ctx context.Context) {
	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		k.applyInterestForCAssetLoans(ctx, CAsset)
	}
}

func (k Keeper) applyInterestForCAssetLoans(ctx context.Context, cAsset *denomtypes.CAsset) {
	utilityRate := k.calculateUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)
	interestRate = interestRate.Quo(math.LegacyNewDecFromInt(math.NewInt(int64(utils.BlocksPerYear))))
	interestRate = interestRate.Add(math.LegacyOneDec())

	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom)
	loanSum.LoanSum = loanSum.LoanSum.Mul(interestRate)
	k.loansSum.Set(ctx, cAsset.BaseDenom, loanSum)
}
