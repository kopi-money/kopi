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

	"github.com/kopi-money/kopi/x/strategies/types"
)

var (
	PrefixParams               = collections.NewPrefix(0)
	PrefixArbitrageDenoms      = collections.NewPrefix(1)
	PrefixAutomations          = collections.NewPrefix(2)
	PrefixAutomationsNextIndex = collections.NewPrefix(3)
	PrefixAutomationFunds      = collections.NewPrefix(4)
)

type (
	Keeper struct {
		cdc    codec.BinaryCodec
		logger log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		// Collections
		params               *cache.ItemCache[types.Params]
		arbitrageDenoms      *cache.MapCache[string, types.ArbitrageDenom]
		automations          *cache.MapCache[uint64, types.Automation]
		automationsNextIndex *cache.ItemCache[uint64]
		automationFunds      *cache.MapCache[string, types.AutomationFunds]

		caches *cache.Caches

		AccountKeeper      types.AccountKeeper
		BankKeeper         types.BankKeeper
		BlockspeedKeeper   types.BlockspeedKeeper
		DistributionKeeper types.DistributionKeeper
		StakingKeeper      types.StakingKeeper

		DenomKeeper types.DenomKeeper
		DexKeeper   types.DexKeeper
		MMKeeper    types.MMKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distributionKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,

	blockspeedKeeper types.BlockspeedKeeper,
	denomKeeper types.DenomKeeper,
	dexKeeper types.DexKeeper,
	mmKeeper types.MMKeeper,

	authority string,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	caches := &cache.Caches{}

	return Keeper{
		cdc:                cdc,
		authority:          authority,
		logger:             logger,
		AccountKeeper:      accountKeeper,
		BankKeeper:         bankKeeper,
		BlockspeedKeeper:   blockspeedKeeper,
		DistributionKeeper: distributionKeeper,
		StakingKeeper:      stakingKeeper,
		DenomKeeper:        denomKeeper,
		DexKeeper:          dexKeeper,
		MMKeeper:           mmKeeper,

		caches: caches,

		params: cache.NewItemCache(
			sb,
			PrefixParams,
			"params",
			codec.CollValue[types.Params](cdc),
			caches,
		),

		arbitrageDenoms: cache.NewMapCache(
			sb,
			PrefixArbitrageDenoms,
			"arbitrage_denoms",
			collections.StringKey,
			codec.CollValue[types.ArbitrageDenom](cdc),
			caches,
		),

		automations: cache.NewMapCache(
			sb,
			PrefixAutomations,
			"automations",
			collections.Uint64Key,
			codec.CollValue[types.Automation](cdc),
			caches,
		),

		automationsNextIndex: cache.NewItemCache(
			sb,
			PrefixAutomationsNextIndex,
			"automations_next_index",
			collections.Uint64Value,
			caches,
		),

		automationFunds: cache.NewMapCache(
			sb,
			PrefixAutomationFunds,
			"automation_funds",
			collections.StringKey,
			codec.CollValue[types.AutomationFunds](cdc),
			caches,
		),
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
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
