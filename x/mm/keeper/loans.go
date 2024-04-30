package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetDenomLoans(ctx context.Context) (denomLoans []types.Loans) {
	for _, denom := range k.DenomKeeper.GetCAssets(ctx) {
		var loans []*types.Loan
		for _, loan := range k.GetAllLoansByDenom(ctx, denom.BaseDenom) {
			loans = append(loans, &loan)
		}

		denomLoans = append(denomLoans, types.Loans{
			Denom: denom.BaseDenom,
			Loans: loans,
		})
	}

	return
}

// GetAllDenomLoans returns all loans
func (k Keeper) GetAllDenomLoans(ctx context.Context) (list []types.Loans) {
	iterator := k.LoanIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Loans
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetLoan set a specific deposits in the store from its index
func (k Keeper) SetLoan(ctx context.Context, denom string, loan types.Loan) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoans))

	// If loan is empty, delete it
	if loan.Amount.LTE(math.LegacyZeroDec()) {
		store.Delete(types.KeyDenomAddress(denom, loan.Address))
		return
	}

	if loan.Index == 0 {
		loan.Index = k.GetNextLoanIndex(ctx)
		k.SetNextLoanIndex(ctx, loan.Index+1)
	}

	b := k.cdc.MustMarshal(&loan)
	store.Set(types.KeyDenomAddress(denom, loan.Address), b)
}

func (k Keeper) GetNextLoanIndex(ctx context.Context) int64 {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoansIndex))

	b := store.Get([]byte{0})
	if b == nil {
		return 0
	}

	var nextIndex types.NextLoanIndex
	k.cdc.MustUnmarshal(b, &nextIndex)
	return nextIndex.Index
}

func (k Keeper) SetNextLoanIndex(ctx context.Context, index int64) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoansIndex))

	b := k.cdc.MustMarshal(&types.NextLoanIndex{Index: index})
	store.Set([]byte{0}, b)
}

// GetLoan returns a deposits from its index
func (k Keeper) GetLoan(ctx context.Context, denom, addess string) (types.Loan, bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoans))

	b := store.Get(types.KeyDenomAddress(denom, addess))
	if b == nil {
		return types.Loan{}, false
	}

	var deposit types.Loan
	k.cdc.MustUnmarshal(b, &deposit)
	return deposit, true
}

func (k Keeper) LoanIterator(ctx context.Context) storetypes.Iterator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoans))
	return storetypes.KVStorePrefixIterator(store, []byte{})
}

func (k Keeper) GetAllLoansByDenom(ctx context.Context, denom string) (list []types.Loan) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoans))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Loan
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetLoansSum(ctx context.Context, denom string) math.LegacyDec {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLoans))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	sum := math.LegacyZeroDec()
	for ; iterator.Valid(); iterator.Next() {
		var val types.Loan
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		sum = sum.Add(val.Amount)
	}

	return sum
}

type CAssetLoan struct {
	types.Loan
	cAsset *denomtypes.CAsset
}

func (k Keeper) getUserLoans(ctx context.Context, address string) (loans []CAssetLoan) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loan, found := k.GetLoan(ctx, cAsset.BaseDenom, address)
		if found {
			loans = append(loans, CAssetLoan{
				Loan:   loan,
				cAsset: cAsset,
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

	iterator := k.LoanIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var loan types.Loan
		k.cdc.MustUnmarshal(iterator.Value(), &loan)

		if _, seen := borrowersMap[loan.Address]; !seen {
			borrowersMap[loan.Address] = struct{}{}
			borrowers = append(borrowers, loan.Address)
		}
	}

	return borrowers
}

func (k Keeper) CalcAvailableToBorrow(ctx context.Context, address, denom string) (math.Int, error) {
	borrowable, err := k.calculateBorrowableAmount(ctx, address, denom)
	if err != nil {
		return math.Int{}, errors.Wrap(err, "could not calculate borrowable amount")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	available := vault.AmountOf(denom)

	return math.MinInt(available, borrowable.TruncateInt()), nil
}
