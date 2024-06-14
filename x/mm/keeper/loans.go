package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

// GetGenesisLoans is used for genesis export
func (k Keeper) GetGenesisLoans(ctx context.Context) (denomLoans []types.Loans) {
	for _, denom := range k.DenomKeeper.GetCAssets(ctx) {
		var loans []*types.Loan
		iterator := k.LoanIterator(ctx, denom.BaseDenom)
		for iterator.Valid() {
			loan := iterator.GetNext()
			loans = append(loans, &loan)
		}

		loanSum := k.GetLoanSumWithDefault(ctx, denom.BaseDenom)
		denomLoans = append(denomLoans, types.Loans{
			Denom:     denom.BaseDenom,
			Loans:     loans,
			WeightSum: loanSum.WeightSum,
			LoanSum:   loanSum.LoanSum,
		})
	}

	return
}

func (k Keeper) loadLoanWithDefault(ctx context.Context, denom, address string) types.Loan {
	loan, has := k.loans.Get(ctx, collections.Join(denom, address))
	if has {
		return loan
	}

	return types.Loan{
		Index:   0,
		Address: address,
		Weight:  math.LegacyZeroDec(),
	}
}

func (k Keeper) SetLoan(ctx context.Context, denom string, loan types.Loan) (uint64, int) {
	key := collections.Join(denom, loan.Address)

	// If loan is empty, delete it
	if loan.Weight.LTE(math.LegacyZeroDec()) {
		k.loans.Remove(ctx, key)
		return loan.Index, -1
	}

	change := 0
	if loan.Index == 0 {
		nextIndex, _ := k.loanNextIndex.Get(ctx)
		loan.Index = nextIndex + 1
		k.loanNextIndex.Set(ctx, nextIndex)
		change = 1
	}

	k.loans.Set(ctx, key, loan)
	return loan.Index, change
}

func (k Keeper) SetNextLoanIndex(ctx context.Context, index uint64) {
	k.loanNextIndex.Set(ctx, index)
}

func (k Keeper) GetNextLoanIndex(ctx context.Context) (uint64, bool) {
	return k.loanNextIndex.Get(ctx)
}

func (k Keeper) LoadLoan(ctx context.Context, denom, address string) (types.Loan, bool) {
	return k.loans.Get(ctx, collections.Join(denom, address))
}

func (k Keeper) LoanIterator(ctx context.Context, denom string) cache.Iterator[collections.Pair[string, string], types.Loan] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	keyPrefix := func(key collections.Pair[string, string]) bool {
		return key.K1() == denom
	}
	return k.loans.Iterator(ctx, rng, keyPrefix)
}

func (k Keeper) GetLoanValue(ctx context.Context, denom, address string) math.LegacyDec {
	key := collections.Join(denom, address)
	loan, found := k.loans.Get(ctx, key)
	if !found {
		return math.LegacyZeroDec()
	}

	loanSum := k.GetLoanSumWithDefault(ctx, denom)
	return k.getLoanValue(loanSum, loan)
}

func (k Keeper) getLoanValue(loanSum types.LoanSum, loan types.Loan) math.LegacyDec {
	if loanSum.WeightSum.Equal(math.LegacyZeroDec()) || loanSum.LoanSum.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	loanValue := loan.Weight.Quo(loanSum.WeightSum).Mul(loanSum.LoanSum)
	return loanValue
}

func (k Keeper) GetLoansNum(ctx context.Context) (num int) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		num += int(k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).NumLoans)
	}

	return
}

func (k Keeper) GetLoansNumForAddress(ctx context.Context, address string) (num int) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		key := collections.Join(cAsset.BaseDenom, address)
		if _, found := k.loans.Get(ctx, key); found {
			num++
		}
	}

	return
}

type CAssetLoan struct {
	types.Loan
	cAsset *denomtypes.CAsset
	value  math.LegacyDec
}

func (k Keeper) getUserLoans(ctx context.Context, address string) (loans []CAssetLoan) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		key := collections.Join(cAsset.BaseDenom, address)
		loan, found := k.loans.Get(ctx, key)
		if found {
			loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom)
			loans = append(loans, CAssetLoan{
				Loan:   loan,
				cAsset: cAsset,
				value:  k.getLoanValue(loanSum, loan),
			})
		}
	}

	return loans
}

type Borrower struct {
	address string
	loans   []CAssetLoan
}

// getBorrowers returns a list with all borrowers and their loans. There might be easier ways to get this, e.g. by using
// maps more prominently, but the intention was to create a function that returns the same list of entries if it is
// called on various nodes.
func (k Keeper) getBorrowers(ctx context.Context) []string {
	var borrowers []string
	borrowersMap := make(map[string]struct{})

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		iterator := k.LoanIterator(ctx, cAsset.BaseDenom)
		for iterator.Valid() {
			loan := iterator.GetNext()
			if _, seen := borrowersMap[loan.Address]; !seen {
				borrowersMap[loan.Address] = struct{}{}
				borrowers = append(borrowers, loan.Address)
			}
		}
	}

	return borrowers
}

func (k Keeper) CalcAvailableToBorrow(ctx context.Context, address, denom string) (math.LegacyDec, error) {
	borrowable, err := k.calculateBorrowableAmount(ctx, address, denom)
	if err != nil {
		return math.LegacyDec{}, errors.Wrap(err, "could not calculate borrowable amount")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	available := vault.AmountOf(denom)

	return math.LegacyMinDec(available.ToLegacyDec(), borrowable), nil
}

func (k Keeper) checkBorrowLimitExceeded(ctx context.Context, cAsset *denomtypes.CAsset, amount math.LegacyDec) bool {
	if cAsset.BorrowLimit.Equal(math.LegacyZeroDec()) {
		return false
	}

	borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum
	deposited := k.calculateCAssetValue(ctx, cAsset)

	borrowLimit := deposited.Mul(cAsset.BorrowLimit)
	return borrowLimit.LT(borrowed.Add(amount))
}

func (k Keeper) updateLoan(ctx context.Context, denom, address string, valueChange math.LegacyDec) (uint64, bool) {
	loanSum := k.GetLoanSumWithDefault(ctx, denom)
	loan := k.loadLoanWithDefault(ctx, denom, address)

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
	k.loansSum.Set(ctx, denom, loanSum)

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

func compareLoans(l1, l2 types.Loan) bool {
	return l1.Weight.Equal(l2.Weight) && l1.Index == l2.Index && l1.Address == l2.Address
}
