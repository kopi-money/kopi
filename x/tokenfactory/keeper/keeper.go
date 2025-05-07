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

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

var (
	PrefixParams                       = collections.NewPrefix(0)
	PrefixDenoms                       = collections.NewPrefix(1)
	PrefixLiquidityPools               = collections.NewPrefix(2)
	PrefixLiquidityProviderShares      = collections.NewPrefix(3)
	PrefixLiquidityUnlockings          = collections.NewPrefix(4)
	PrefixLiquidityUnlockingsNextIndex = collections.NewPrefix(5)
	PrefixOffers                       = collections.NewPrefix(6)
	PrefixOffersNextIndex              = collections.NewPrefix(7)
	PrefixVestings                     = collections.NewPrefix(8)
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

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		// Collections
		params                       *cache.ItemCache[types.Params]
		factoryDenoms                *cache.MapCache[string, types.FactoryDenom]
		liquidityPools               *cache.MapCache[string, types.LiquidityPool]
		liquidityProviderShares      *cache.NestedMapCache[string, string, types.ProviderShare]
		liquidityUnlockings          *cache.MapCache[uint64, types.LiquidityUnlocking]
		liquidityUnlockingsNextIndex *cache.ItemCache[uint64]
		offers                       *cache.MapCache[uint64, types.Offer]
		offersNextIndex              *cache.ItemCache[uint64]
		vestings                     *cache.MapCache[uint64, types.Vesting]
		vestingsNextIndex            *cache.ItemCache[uint64]

		caches *cache.Caches
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
		logger:        logger,
		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		DenomKeeper:   denomKeeper,
		DexKeeper:     dexKeeper,

		caches: caches,

		params: cache.NewItemCache(
			sb,
			PrefixParams,
			"params",
			codec.CollValue[types.Params](cdc),
			caches,
		),

		factoryDenoms: cache.NewMapCache(
			sb,
			PrefixDenoms,
			"denoms",
			collections.StringKey,
			codec.CollValue[types.FactoryDenom](cdc),
			caches,
		),

		liquidityPools: cache.NewMapCache(
			sb,
			PrefixLiquidityPools,
			"liquidity_pools",
			collections.StringKey,
			codec.CollValue[types.LiquidityPool](cdc),
			caches,
		),

		liquidityProviderShares: cache.NewNestedMapCache(
			sb,
			PrefixLiquidityProviderShares,
			"liquidity_provider_shares",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.ProviderShare](cdc),
			caches,
		),

		liquidityUnlockings: cache.NewMapCache(
			sb,
			PrefixLiquidityUnlockings,
			"liquidity_unlockings",
			collections.Uint64Key,
			codec.CollValue[types.LiquidityUnlocking](cdc),
			caches,
		),

		liquidityUnlockingsNextIndex: cache.NewItemCache(
			sb,
			PrefixLiquidityUnlockingsNextIndex,
			"liquidity_unlockings_next_index",
			collections.Uint64Value,
			caches,
		),

		offers: cache.NewMapCache(
			sb,
			PrefixOffers,
			"offers",
			collections.Uint64Key,
			codec.CollValue[types.Offer](cdc),
			caches,
		),

		offersNextIndex: cache.NewItemCache(
			sb,
			PrefixOffersNextIndex,
			"offers_next_index",
			collections.Uint64Value,
			caches,
		),

		vestings: cache.NewMapCache(
			sb,
			PrefixVestings,
			"vestings",
			collections.Uint64Key,
			codec.CollValue[types.Vesting](cdc),
			caches,
		),

		vestingsNextIndex: cache.NewItemCache(
			sb,
			PrefixOffersNextIndex,
			"vestings_next_index",
			collections.Uint64Value,
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
