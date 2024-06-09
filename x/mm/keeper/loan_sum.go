package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetLoanSumWithDefault(ctx context.Context, denom string) types.LoanSum {
	loanSum, has := k.loansSum.Get(ctx, denom)
	if has {
		return loanSum
	}

	return types.LoanSum{
		Denom:     denom,
		NumLoans:  0,
		LoanSum:   math.LegacyZeroDec(),
		WeightSum: math.LegacyZeroDec(),
	}
}

func (k Keeper) SetLoanSum(ctx context.Context, sum types.LoanSum) {
	k.loansSum.Set(ctx, sum.Denom, sum)
}

func compareLoanSums(ls1, ls2 types.LoanSum) bool {
	return ls1.WeightSum.Equal(ls2.WeightSum) &&
		ls1.LoanSum.Equal(ls2.LoanSum) &&
		ls1.Denom == ls2.Denom &&
		ls1.NumLoans == ls2.NumLoans
}
