package keeper

import (
	"context"
	"testing"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/runtime"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	reservekeeper "github.com/kopi-money/kopi/x/reserve/keeper"
	swapkeeper "github.com/kopi-money/kopi/x/swap/keeper"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
)

func SwapKeeper(t *testing.T) (swapkeeper.Keeper, dexkeeper.Keeper, reservekeeper.Keeper, context.Context) {
	dexKeeper, ctx, keys := DexKeeper(t)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	swapKeeper := swapkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys.swp),
		log.NewNopLogger(),
		dexKeeper.AccountKeeper,
		dexKeeper.BankKeeper,
		DummyBlockspeedKeeper{},
		dexKeeper.DenomKeeper.(swaptypes.DenomKeeper),
		dexKeeper,
		authority.String(),
	)
	cache.AddCache(swapKeeper)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return swapKeeper.SetParams(innerCtx, swaptypes.DefaultParams())
	}))

	accountKeeper := swapKeeper.AccountKeeper.(authkeeper.AccountKeeper)
	swapAcc := authtypes.NewEmptyModuleAccount(swaptypes.ModuleName, authtypes.Burner, authtypes.Minter)
	acc := accountKeeper.NewAccountWithAddress(ctx, sdk.AccAddress(swapAcc.Address))
	require.NoError(t, swapAcc.SetAccountNumber(acc.GetAccountNumber()))
	accountKeeper.SetAccount(ctx, swapAcc)

	reserveKeeper := reservekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys.res),
		log.NewNopLogger(),
		accountKeeper,
		dexKeeper.BankKeeper,
		dexKeeper.DenomKeeper,
		dexKeeper,
		NewMMKeeper(keys, dexKeeper, authority),
		authority.String())

	return swapKeeper, dexKeeper, reserveKeeper, ctx
}

func SetupSwapMsgServer(t *testing.T) (swapkeeper.Keeper, dextypes.MsgServer, dexkeeper.Keeper, reservekeeper.Keeper, context.Context) {
	k, dexKeeper, reserveKeeper, ctx := SwapKeeper(t)
	addFunds(ctx, dexKeeper.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, dexkeeper.NewMsgServerImpl(dexKeeper), dexKeeper, reserveKeeper, ctx
}
