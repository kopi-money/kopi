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
	sum.LoanSum = math.LegacyMaxDec(sum.LoanSum, math.LegacyZeroDec())
	sum.WeightSum = math.LegacyMaxDec(sum.WeightSum, math.LegacyZeroDec())

	k.loansSum.Set(ctx, sum.Denom, sum)
}
