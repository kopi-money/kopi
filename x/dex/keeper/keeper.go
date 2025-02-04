package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

var (
	PrefixParams             = collections.NewPrefix(0)
	PrefixLiquidityEntries   = collections.NewPrefix(1)
	PrefixLiquidityNextIndex = collections.NewPrefix(2)
	PrefixOrdersLegacy       = collections.NewPrefix(3)
	PrefixOrdersNextIndex    = collections.NewPrefix(4)
	PrefixTradeAmounts       = collections.NewPrefix(5)
	PrefixBaseTradeFee       = collections.NewPrefix(6)
	PrefixOrders             = collections.NewPrefix(7)
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
		liquidityEntries          *cache.NestedMapCache[string, uint64, types.Liquidity]
		liquidityEntriesNextIndex *cache.ItemCache[uint64]
		orders                    *cache.MapCache[uint64, types.Order]
		ordersLegacy              *cache.MapCache[uint64, types.LegacyOrder]
		ordersNextIndex           *cache.ItemCache[uint64]
		tradeAmounts              *cache.MapCache[string, types.WalletTradeAmount]
		tradeFeeTracker           *cache.ItemCache[int64]

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
			sb,
			PrefixParams,
			"params",
			codec.CollValue[types.Params](cdc),
			caches,
		),

		liquidityEntries: cache.NewNestedMapCache(
			sb,
			PrefixLiquidityEntries,
			"liquidity_entries",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
			codec.CollValue[types.Liquidity](cdc),
			caches,
		),

		liquidityEntriesNextIndex: cache.NewItemCache(
			sb,
			PrefixLiquidityNextIndex,
			"liquidity_entries_next_index",
			collections.Uint64Value,
			caches,
		),

		ordersLegacy: cache.NewMapCache(
			sb,
			PrefixOrdersLegacy,
			"orders_list",
			collections.Uint64Key,
			codec.CollValue[types.LegacyOrder](cdc),
			caches,
		),

		orders: cache.NewMapCache(
			sb,
			PrefixOrders,
			"orders_list_v2",
			collections.Uint64Key,
			codec.CollValue[types.Order](cdc),
			caches,
		),

		ordersNextIndex: cache.NewItemCache(
			sb,
			PrefixOrdersNextIndex,
			"orders_next_index",
			collections.Uint64Value,
			caches,
		),

		tradeAmounts: cache.NewMapCache(
			sb,
			PrefixTradeAmounts,
			"trade_amounts",
			collections.StringKey,
			codec.CollValue[types.WalletTradeAmount](cdc),
			caches,
		),

		tradeFeeTracker: cache.NewItemCache(
			sb,
			PrefixBaseTradeFee,
			"trade_fee_tracker",
			collections.Int64Value,
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
