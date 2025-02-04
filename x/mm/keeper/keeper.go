package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/types"
)

var (
	PrefixParams        = collections.NewPrefix(0)
	PrefixCollateral    = collections.NewPrefix(1)
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

		AccountKeeper    types.AccountKeeper
		BlockspeedKeeper types.BlockspeedKeeper
		BankKeeper       types.BankKeeper
		DenomKeeper      types.DenomKeeper
		DexKeeper        types.DexKeeper

		// Collections
		params        *cache.ItemCache[types.Params]
		collateral    *cache.NestedMapCache[string, string, types.Collateral]
		loans         *cache.NestedMapCache[string, string, types.Loan]
		loanNextIndex *cache.ItemCache[uint64]
		loansSum      *cache.MapCache[string, types.LoanSum]
		redemptions   *cache.NestedMapCache[string, string, types.Redemption]

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
	blockspeedKeeper types.BlockspeedKeeper,
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
		cdc:              cdc,
		storeService:     storeService,
		authority:        authority,
		AccountKeeper:    accountKeeper,
		BankKeeper:       bankKeeper,
		BlockspeedKeeper: blockspeedKeeper,
		DenomKeeper:      denomKeeper,
		DexKeeper:        dexKeeper,
		logger:           logger,

		caches: caches,

		params: cache.NewItemCache(
			sb,
			PrefixParams,
			"params",
			codec.CollValue[types.Params](cdc),
			caches,
		),

		collateral: cache.NewNestedMapCache(
			sb,
			PrefixCollateral,
			"collateral_entries",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.Collateral](cdc),
			caches,
		),

		loans: cache.NewNestedMapCache(
			sb,
			PrefixLoans,
			"loans_list",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.Loan](cdc),
			caches,
		),

		loanNextIndex: cache.NewItemCache(
			sb,
			PrefixLoanNextIndex,
			"loans_next_index",
			collections.Uint64Value,
			caches,
		),

		loansSum: cache.NewMapCache(
			sb,
			PrefixLoanSum,
			"loans_sum",
			collections.StringKey,
			codec.CollValue[types.LoanSum](cdc),
			caches,
		),

		redemptions: cache.NewNestedMapCache(
			sb,
			PrefixRedemptions,
			"redemptions",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.Redemption](cdc),
			caches,
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

func (k Keeper) Rollback(ctx context.Context) {
	k.caches.Rollback(ctx)
}

func (k Keeper) CommitToCache(ctx context.Context) {
	k.caches.CommitToCache(ctx)
}

func (k Keeper) Clear(ctx context.Context) {
	k.caches.Clear(ctx)
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
