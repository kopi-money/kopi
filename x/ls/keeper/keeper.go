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

	"github.com/kopi-money/kopi/x/ls/types"
)

var (
	PrefixParams                 = collections.NewPrefix(0)
	PrefixUndelegations          = collections.NewPrefix(1)
	PrefixUndelegationsNextIndex = collections.NewPrefix(2)
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		accountKeeper      types.AccountKeeper
		bankKeeper         types.BankKeeper
		distributionKeeper types.DistributionKeeper
		stakingKeeper      types.StakingKeeper

		// Collections
		params                 *cache.ItemCache[types.Params]
		undelegations          *cache.MapCache[uint64, types.Undelegation]
		undelegationsNextIndex *cache.ItemCache[uint64]

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
	distributionKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
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

		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		distributionKeeper: distributionKeeper,
		stakingKeeper:      stakingKeeper,

		params: cache.NewItemCache(
			sb,
			PrefixParams,
			"params",
			codec.CollValue[types.Params](cdc),
			caches,
		),

		undelegations: cache.NewMapCache(
			sb,
			PrefixUndelegations,
			"undelegations",
			collections.Uint64Key,
			codec.CollValue[types.Undelegation](cdc),
			caches,
		),

		undelegationsNextIndex: cache.NewItemCache(
			sb,
			PrefixUndelegationsNextIndex,
			"undelegations_next_index",
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
