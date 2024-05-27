package keeper

import (
	"context"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetLoanSum(ctx context.Context, denom string) types.LoanSum {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoanSums))

	b := store.Get(types.KeyDenom(denom))
	if b == nil {
		return types.LoanSum{
			Denom:     denom,
			LoanSum:   math.LegacyZeroDec(),
			WeightSum: math.LegacyZeroDec(),
		}
	}

	var loanSum types.LoanSum
	k.cdc.MustUnmarshal(b, &loanSum)
	return loanSum
}

func (k Keeper) SetLoanSum(ctx context.Context, loanSum types.LoanSum) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoanSums))
	b := k.cdc.MustMarshal(&loanSum)
	store.Set(types.KeyDenom(loanSum.Denom), b)
}

func (k Keeper) updateLoan(ctx context.Context, denom, address string, valueChange math.LegacyDec) (int64, bool) {
	loanSum := k.GetLoanSum(ctx, denom)
	loan, _ := k.GetLoan(ctx, denom, address)

	// First, fully remove the loan from the loan sum
	loanValue := calculateLoanValue(loanSum, loan.Weight)
	loanSum.WeightSum = loanSum.WeightSum.Sub(loan.Weight)
	loanSum.LoanSum = loanSum.LoanSum.Sub(loanValue)

	// Calculate the new weight
	newLoanValue := loanValue.Add(valueChange)
	newWeight := calculateLoanWeight(loanSum, newLoanValue)

	// Update loan and loanSum
	loan.Weight = newWeight
	loanSum.WeightSum = loanSum.WeightSum.Add(newWeight)
	loanSum.LoanSum = loanSum.LoanSum.Add(newLoanValue)

	loanIndex, numLoanChange := k.SetLoan(ctx, denom, loan)
	loanSum.NumLoans += uint64(numLoanChange)
	k.SetLoanSum(ctx, loanSum)

	return loanIndex, numLoanChange == -1
}

func calculateLoanValue(loanSum types.LoanSum, weight math.LegacyDec) math.LegacyDec {
	if loanSum.WeightSum.Equal(math.LegacyZeroDec()) || weight.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	share := weight.Quo(loanSum.WeightSum)
	return loanSum.LoanSum.Mul(share)
}

func calculateLoanWeight(loanSum types.LoanSum, addedAmount math.LegacyDec) math.LegacyDec {
	newLoanSum := loanSum.LoanSum.Add(addedAmount)

	var valueShare math.LegacyDec
	if newLoanSum.GT(math.LegacyZeroDec()) {
		valueShare = addedAmount.Quo(newLoanSum)
	} else {
		valueShare = math.LegacyZeroDec()
	}

	var additionalWeight math.LegacyDec
	if valueShare.Equal(math.LegacyOneDec()) {
		additionalWeight = addedAmount
	} else {
		additionalWeight = loanSum.WeightSum.Quo(math.LegacyOneDec().Sub(valueShare)).Sub(loanSum.WeightSum)
	}

	return additionalWeight
}
