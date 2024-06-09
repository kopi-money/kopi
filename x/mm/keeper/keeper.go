package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"fmt"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/types"
)

var (
	PrefixCollateral    = collections.NewPrefix(0)
	PrefixCollateralSum = collections.NewPrefix(1)
	PrefixLoans         = collections.NewPrefix(2)
	PrefixLoanNextIndex = collections.NewPrefix(3)
	PrefixLoanSum       = collections.NewPrefix(4)
	PrefixRedemptions   = collections.NewPrefix(5)
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		AccountKeeper types.AccountKeeper
		BankKeeper    types.BankKeeper
		DenomKeeper   types.DenomKeeper
		DexKeeper     types.DexKeeper

		// Collections
		collateral    *cache.MapCache[collections.Pair[string, string], types.Collateral]
		collateralSum *cache.MapCache[string, types.CollateralSum]
		loans         *cache.MapCache[collections.Pair[string, string], types.Loan]
		loanNextIndex *cache.ItemCache[uint64]
		loansSum      *cache.MapCache[string, types.LoanSum]
		redemptions   *cache.MapCache[collections.Pair[string, string], types.Redemption]

		// Caches
		caches *cache.Caches

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	denomKeeper types.DenomKeeper,
	dexKeeper types.DexKeeper,
	authority string,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	caches := &cache.Caches{}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		DenomKeeper:   denomKeeper,
		DexKeeper:     dexKeeper,
		logger:        logger,

		caches: caches,

		collateral: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixCollateral,
				"collateral_entries",
				collections.PairKeyCodec(collections.StringKey, collections.StringKey),
				codec.CollValue[types.Collateral](cdc),
			),
			caches,
			cache.StringStringComparer,
			compareCollaterals,
		),

		collateralSum: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixCollateralSum,
				"collateral_sum",
				collections.StringKey,
				codec.CollValue[types.CollateralSum](cdc),
			),
			caches,
			cache.StringComparer,
			compareCollateralSums,
		),

		loans: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixLoans,
				"loans_list",
				collections.PairKeyCodec(collections.StringKey, collections.StringKey),
				codec.CollValue[types.Loan](cdc),
			),
			caches,
			cache.StringStringComparer,
			compareLoans,
		),

		loanNextIndex: cache.NewItemCache(
			collections.NewItem(
				sb,
				PrefixLoanNextIndex,
				"loans_next_index",
				collections.Uint64Value,
			),
			caches,
			cache.ValueComparerUint64,
		),

		loansSum: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixLoanSum,
				"loans_sum",
				collections.StringKey,
				codec.CollValue[types.LoanSum](cdc),
			),
			caches,
			cache.StringComparer,
			compareLoanSums,
		),

		redemptions: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixRedemptions,
				"redemptions",
				collections.PairKeyCodec(collections.StringKey, collections.StringKey),
				codec.CollValue[types.Redemption](cdc),
			),
			caches,
			cache.StringStringComparer,
			compareRedemptions,
		),
	}
}

func (k Keeper) Initialize(ctx context.Context) error {
	return k.caches.Initialize(ctx)
}

func (k Keeper) CommitToDB(ctx context.Context) error {
	return k.caches.CommitToDB(ctx)
}

func (k Keeper) CheckCache(ctx context.Context) error {
	return k.caches.CheckCache(ctx)
}

func (k Keeper) CommitToCache(ctx context.Context) {
	k.caches.CommitToCache(ctx)
}

func (k Keeper) Rollback(ctx context.Context) {
	k.caches.Rollback(ctx)
}

func (k Keeper) ClearTransactions() {
	k.caches.ClearTransactions()
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
