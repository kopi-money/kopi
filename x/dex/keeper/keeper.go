package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"
)

var (
	PrefixParams             = collections.NewPrefix(0)
	PrefixLiquiditySums      = collections.NewPrefix(1)
	PrefixLiquidityPairs     = collections.NewPrefix(2)
	PrefixLiquidityEntries   = collections.NewPrefix(3)
	PrefixLiquidityNextIndex = collections.NewPrefix(4)
	PrefixOrders             = collections.NewPrefix(5)
	PrefixOrdersNextIndex    = collections.NewPrefix(6)
	PrefixRatios             = collections.NewPrefix(7)
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		AccountKeeper types.AccountKeeper
		DenomKeeper   types.DenomKeeper
		BankKeeper    types.BankKeeper

		// Collections
		params                    *cache.ItemCache[types.Params]
		liquiditySums             *cache.MapCache[string, types.LiquiditySum]
		liquidityPairs            *cache.MapCache[string, types.LiquidityPair]
		liquidityEntries          *cache.MapCache[collections.Pair[string, uint64], types.Liquidity]
		liquidityEntriesNextIndex *cache.ItemCache[uint64]
		orders                    *cache.MapCache[uint64, types.Order]
		ordersNextIndex           *cache.ItemCache[uint64]
		ratios                    *cache.MapCache[string, types.Ratio]

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
	authority string,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	caches := &cache.Caches{}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		DenomKeeper:   denomKeeper,

		caches: caches,

		params: cache.NewItemCache(
			collections.NewItem(
				sb,
				PrefixParams,
				"params",
				codec.CollValue[types.Params](cdc),
			),
			caches,
			func(v1, v2 types.Params) bool { return true },
		),

		liquiditySums: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixLiquiditySums,
				"liquidity_sums",
				collections.StringKey,
				codec.CollValue[types.LiquiditySum](cdc),
			),
			caches,
			cache.StringComparer,
			compareLiquiditySums,
		),

		liquidityPairs: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixLiquidityPairs,
				"liquidity_pairs",
				collections.StringKey,
				codec.CollValue[types.LiquidityPair](cdc),
			),
			caches,
			cache.StringComparer,
			compareLiquidityPairs,
		),

		liquidityEntries: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixLiquidityEntries,
				"liquidity_entries",
				collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
				codec.CollValue[types.Liquidity](cdc),
			),
			caches,
			cache.StringUInt64Comparer,
			compareLiquidity,
		),

		liquidityEntriesNextIndex: cache.NewItemCache(
			collections.NewItem(
				sb,
				PrefixLiquidityNextIndex,
				"liquidity_entries_next_index",
				collections.Uint64Value,
			),
			caches,
			cache.ValueComparerUint64,
		),

		orders: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixOrders,
				"orders_list",
				collections.Uint64Key,
				codec.CollValue[types.Order](cdc),
			),
			caches,
			cache.Uint64Comparer,
			compareOrders,
		),

		ordersNextIndex: cache.NewItemCache(
			collections.NewItem(
				sb,
				PrefixOrdersNextIndex,
				"orders_next_index",
				collections.Uint64Value,
			),
			caches,
			cache.ValueComparerUint64,
		),

		ratios: cache.NewCacheMap(
			collections.NewMap(
				sb,
				PrefixRatios,
				"ratios",
				collections.StringKey,
				codec.CollValue[types.Ratio](cdc),
			),
			caches,
			cache.StringComparer,
			compareRatios,
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
