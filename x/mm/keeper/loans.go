package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

// GetGenesisLoans is used for genesis export
func (k Keeper) GetGenesisLoans(ctx context.Context) (denomLoans []types.Loans) {
	for _, denom := range k.DenomKeeper.GetCAssets(ctx) {
		var loans []types.GenesisLoan
		iterator := k.LoanIterator(ctx, denom.BaseDexDenom)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()
			loan := keyValue.Value().Value()

			loans = append(loans, types.GenesisLoan{
				Index:   loan.Index,
				Address: keyValue.Key(),
				Weight:  loan.Weight,
			})
		}

		loanSum := k.GetLoanSumWithDefault(ctx, denom.BaseDexDenom)
		denomLoans = append(denomLoans, types.Loans{
			Denom:     denom.BaseDexDenom,
			Loans:     loans,
			WeightSum: loanSum.WeightSum,
			LoanSum:   loanSum.LoanSum,
		})
	}

	return
}

func (k Keeper) loadLoanWithDefault(ctx context.Context, denom, address string) types.Loan {
	loan, has := k.loans.Get(ctx, denom, address)
	if has {
		return loan
	}

	return types.Loan{
		Index:  0, // Index 0 indicates this is a new loan
		Weight: math.LegacyZeroDec(),
	}
}

func (k Keeper) SetLoan(ctx context.Context, denom, address string, loan types.Loan) (uint64, int) {
	change := 0

	// If loan is empty, delete it
	if !loan.Weight.IsPositive() {
		if has := k.loans.Has(ctx, denom, address); has {
			k.loans.Remove(ctx, denom, address)
			change = -1
		}

		return loan.Index, change
	}

	if loan.Index == 0 {
		nextIndex, _ := k.loanNextIndex.Get(ctx)
		nextIndex += 1
		k.loanNextIndex.Set(ctx, nextIndex)
		loan.Index = nextIndex

		change = 1
	}

	k.loans.Set(ctx, denom, address, loan)
	return loan.Index, change
}

func (k Keeper) SetNextLoanIndex(ctx context.Context, index uint64) {
	k.loanNextIndex.Set(ctx, index)
}

func (k Keeper) GetNextLoanIndex(ctx context.Context) (uint64, bool) {
	return k.loanNextIndex.Get(ctx)
}

func (k Keeper) LoadLoan(ctx context.Context, denom, address string) (types.Loan, bool) {
	return k.loans.Get(ctx, denom, address)
}

func (k Keeper) LoanIterator(ctx context.Context, denom string) cache.Iterator[string, types.Loan] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	return k.loans.Iterator(ctx, rng, denom)
}

func (k Keeper) GetLoanValue(ctx context.Context, denom, address string) math.LegacyDec {
	loan, found := k.loans.Get(ctx, denom, address)
	if !found {
		return math.LegacyZeroDec()
	}

	loanSum := k.GetLoanSumWithDefault(ctx, denom)
	return k.getLoanValue(loanSum, loan)
}

func (k Keeper) getLoanValue(loanSum types.LoanSum, loan types.Loan) math.LegacyDec {
	if loanSum.WeightSum.IsZero() || loanSum.LoanSum.IsZero() {
		return math.LegacyZeroDec()
	}

	if loan.Weight.IsNil() || !loan.Weight.IsPositive() {
		return math.LegacyZeroDec()
	}

	loanValue := loan.Weight.Quo(loanSum.WeightSum).Mul(loanSum.LoanSum) // C
	return loanValue
}

func (k Keeper) GetLoansNum(ctx context.Context) (num int) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		num += int(k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).NumLoans)
	}

	return
}

func (k Keeper) GetLoansNumForAddress(ctx context.Context, address string) (num int) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		if _, found := k.loans.Get(ctx, cAsset.BaseDexDenom, address); found {
			num++
		}
	}

	return
}

type CAssetLoan struct {
	types.Loan
	cAsset denomtypes.CAsset
	value  math.LegacyDec
}

func (k Keeper) getUserLoans(ctx context.Context, address string) (loans []CAssetLoan) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loan, found := k.loans.Get(ctx, cAsset.BaseDexDenom, address)
		if found {
			loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom)
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

// getBorrowers returns a list with all borrowers and their loans. By iterating over all loans, the list of borrowers
// automatically is sorted by loan age: Borrowers with the oldest loan are added first, borrowers with the younger
// loan are added later.
func (k Keeper) getBorrowers(ctx context.Context) (borrowers []string) {
	borrowersMap := make(map[string]uint64)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		iterator := k.LoanIterator(ctx, cAsset.BaseDexDenom)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()
			loan := keyValue.Value().Value()

			lowestIndex, seen := borrowersMap[keyValue.Key()]
			if !seen || lowestIndex < borrowersMap[keyValue.Key()] {
				borrowersMap[keyValue.Key()] = loan.Index
			}
		}
	}

	return rankMapStringInt(borrowersMap)
}

// CalcAvailableToBorrow returns the amount of funds available for borrowing given an address and borrowable denom. It
// checks the theoretical amount given the provided collateral as well as what is available for borrowing given the
// borrow denom's available funds and borrow limit. The smaller value of the two is returned.
func (k Keeper) CalcAvailableToBorrow(ctx context.Context, address, denom string) (math.Int, error) {
	borrowable, err := k.CalculateBorrowableAmount(ctx, address, denom)
	if err != nil {
		return math.Int{}, fmt.Errorf("could not calculate borrowable amount: %w", err)
	}

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, denom)
	if err != nil {
		return math.Int{}, fmt.Errorf("could not get c asset asset: %w", err)
	}

	available := k.availableToBorrowForDenom(ctx, cAsset)

	return math.MinInt(available, borrowable.TruncateInt()), nil
}

func (k Keeper) checkBorrowLimitExceeded(ctx context.Context, cAsset denomtypes.CAsset, amount math.Int) bool {
	if cAsset.BorrowLimit.IsZero() {
		return false
	}

	borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
	deposited := k.CalculateCAssetValue(ctx, cAsset)

	borrowLimit := deposited.Mul(cAsset.BorrowLimit)
	return borrowLimit.LT(borrowed.Add(amount.ToLegacyDec()))
}

// availableToBorrowForDenom returns the amount of funds that are available to borrow for a denom given its borrow
// limit. It multiplies the cAsset value, ie the loans and the funds in the vault, with the borrow limit to get the
// theoretical limit. Then it substracts the amount of loans taken out to get the amount of funds below the borrow limit.
func (k Keeper) availableToBorrowForDenom(ctx context.Context, cAsset denomtypes.CAsset) math.Int {
	borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
	cAssetValue := k.CalculateCAssetValue(ctx, cAsset)
	borrowLimit := cAssetValue.Mul(cAsset.BorrowLimit)
	availableToBorrow := borrowLimit.Sub(borrowed)
	availableToBorrow = math.LegacyMaxDec(math.LegacyZeroDec(), availableToBorrow)
	return availableToBorrow.TruncateInt()
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
	loan.Weight = calculateLoanWeight(loanSum, newLoanValue)

	// Update loan and loanSum
	loanSum.WeightSum = loanSum.WeightSum.Add(loan.Weight)
	loanSum.LoanSum = loanSum.LoanSum.Add(newLoanValue)

	loanIndex, numLoanChange := k.SetLoan(ctx, denom, address, loan)

	if numLoanChange != 0 {
		loanSum.NumLoans += uint64(numLoanChange)
	}

	if loanSum.NumLoans < 0 {
		loanSum.NumLoans = 0
	}

	k.SetLoanSum(ctx, loanSum)
	return loanIndex, numLoanChange == -1
}

func calculateLoanValue(loanSum types.LoanSum, weight math.LegacyDec) math.LegacyDec {
	if loanSum.WeightSum.IsZero() || weight.IsZero() {
		return math.LegacyZeroDec()
	}

	valueShare := weight.Quo(loanSum.WeightSum) // C
	return loanSum.LoanSum.Mul(valueShare)
}

func calculateLoanWeight(loanSum types.LoanSum, addedAmount math.LegacyDec) math.LegacyDec {
	newLoanSum := loanSum.LoanSum.Add(addedAmount)

	var valueShare math.LegacyDec
	if newLoanSum.IsPositive() {
		valueShare = addedAmount.Quo(newLoanSum) // C
	} else {
		valueShare = math.LegacyZeroDec()
	}

	var additionalWeight math.LegacyDec
	if valueShare.Equal(math.LegacyOneDec()) || loanSum.WeightSum.IsZero() {
		additionalWeight = addedAmount
	} else {
		additionalWeight = loanSum.WeightSum.Quo(math.LegacyOneDec().Sub(valueShare)).Sub(loanSum.WeightSum) // C
	}

	return additionalWeight
}

func rankMapStringInt(values map[string]uint64) []string {
	type kv struct {
		Key   string
		Value uint64
	}

	var ss []kv
	for k, v := range values {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value < ss[j].Value
	})

	ranked := make([]string, len(values))
	for i, kv := range ss {
		ranked[i] = kv.Key
	}

	return ranked
}
