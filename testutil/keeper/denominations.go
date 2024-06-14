package keeper

import (
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"
)

type Keys struct {
	cdc      *codec.ProtoCodec
	registry codectypes.InterfaceRegistry

	acc *storetypes.KVStoreKey
	dex *storetypes.KVStoreKey
	bnk *storetypes.KVStoreKey
	dnm *storetypes.KVStoreKey
	mm  *storetypes.KVStoreKey
	swp *storetypes.KVStoreKey
}

func DenomKeeper(t *testing.T) (denomkeeper.Keeper, sdk.Context, *Keys) {
	initSDKConfig()
	cache.NewTranscationHandler()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	keys := Keys{
		acc: storetypes.NewKVStoreKey(authtypes.StoreKey),
		bnk: storetypes.NewKVStoreKey(banktypes.StoreKey),
		dex: storetypes.NewKVStoreKey(dextypes.StoreKey),
		dnm: storetypes.NewKVStoreKey(denomtypes.StoreKey),
		mm:  storetypes.NewKVStoreKey(mmtypes.StoreKey),
		swp: storetypes.NewKVStoreKey(swaptypes.StoreKey),

		cdc:      cdc,
		registry: registry,
	}

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(keys.acc, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys.bnk, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys.dex, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys.dnm, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys.mm, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys.swp, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	require.NoError(t, stateStore.LoadLatestVersion())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	denomKeeper := denomkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys.dnm),
		log.NewNopLogger(),
		authority.String(),
	)
	cache.AddCache(denomKeeper)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	params := denomtypes.DefaultParams()

	factor := math.LegacyNewDec(10)
	params.DexDenoms = append(params.DexDenoms, &denomtypes.DexDenom{
		Name:         "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5",
		Factor:       &factor,
		MinLiquidity: math.NewInt(100_000),
		MinOrderSize: math.NewInt(1_000_000),
	})

	require.NoError(t, cache.Transact(ctx, func(innerCntext sdk.Context) error {
		return denomKeeper.SetParams(innerCntext, params)
	}))

	return denomKeeper, ctx, &keys
}
