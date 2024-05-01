package keeper

import (
	"context"
	"strconv"
	"testing"

	"cosmossdk.io/log"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dexmodule "github.com/kopi-money/kopi/x/dex/module"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"
)

func DexKeeper(t *testing.T) (dexkeeper.Keeper, sdk.Context, *Keys) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		govtypes.ModuleName:        {authtypes.Burner},
		distrtypes.ModuleName:      nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
		dextypes.PoolLiquidity: nil,
		dextypes.PoolTrade:     nil,
		dextypes.PoolFees:      nil,
		dextypes.PoolOrders:    nil,
		dextypes.PoolReserve:   {authtypes.Minter, authtypes.Burner},
		denomtypes.ModuleName:  nil,
		minttypes.ModuleName:   nil,
		mmtypes.PoolCollateral: nil,
		mmtypes.PoolVault:      nil,
		mmtypes.PoolRedemption: nil,
		mmtypes.ModuleName:     {authtypes.Minter},
		swaptypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
	}

	blackListAddrs := map[string]bool{
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
	}

	denomKeeper, ctx, keys := DenomKeeper(t)

	accountKeeper := authkeeper.NewAccountKeeper(
		keys.cdc,
		runtime.NewKVStoreService(keys.acc),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		"kopi",
		authority.String(),
	)

	authtypes.RegisterInterfaces(keys.registry)
	denomtypes.RegisterInterfaces(keys.registry)
	dextypes.RegisterInterfaces(keys.registry)

	bankKeeper := bankkeeper.NewBaseKeeper(
		keys.cdc,
		runtime.NewKVStoreService(keys.bnk),
		accountKeeper,
		blackListAddrs,
		authority.String(),
		log.NewNopLogger(),
	)

	dexKeeper := keeper.NewKeeper(
		keys.cdc,
		runtime.NewKVStoreService(keys.dex),
		log.NewNopLogger(),
		accountKeeper,
		bankKeeper,
		denomKeeper,
		authority.String(),
	)

	require.NoError(t, dexKeeper.SetParams(ctx, dextypes.DefaultParams()))
	dexKeeper.InitPairs(ctx)

	gs := dextypes.DefaultGenesis()
	dexmodule.InitGenesis(ctx, dexKeeper, *gs)

	for _, addr := range []string{Alice, Bob, Carol} {
		accAddr, err := sdk.AccAddressFromBech32(addr)
		require.NoError(t, err)
		acc := accountKeeper.NewAccountWithAddress(ctx, accAddr)
		accountKeeper.SetAccount(ctx, acc)
	}

	reserveAcc := authtypes.NewEmptyModuleAccount(dextypes.PoolReserve, authtypes.Burner, authtypes.Minter)
	acc := accountKeeper.NewAccountWithAddress(ctx, sdk.AccAddress(reserveAcc.Address))
	require.NoError(t, reserveAcc.SetAccountNumber(acc.GetAccountNumber()))
	accountKeeper.SetAccount(ctx, reserveAcc)

	return dexKeeper, ctx, keys
}

func AddLiquidity(ctx context.Context, k dextypes.MsgServer, address, denom string, amount int64) error {
	add := dextypes.MsgAddLiquidity{
		Creator: address,
		Denom:   denom,
		Amount:  strconv.Itoa(int(amount)),
	}
	_, err := k.AddLiquidity(ctx, &add)
	return err
}

func RemoveLiquidity(ctx context.Context, k dextypes.MsgServer, address, denom string, amount int64) error {
	rem := dextypes.MsgRemoveLiquidity{
		Creator: address,
		Denom:   denom,
		Amount:  strconv.Itoa(int(amount)),
	}
	_, err := k.RemoveLiquidity(ctx, &rem)
	return err
}

func SetupDexMsgServer(t *testing.T) (dexkeeper.Keeper, dextypes.MsgServer, sdk.Context) {
	k, ctx, _ := DexKeeper(t)
	addFunds(ctx, k.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, dexkeeper.NewMsgServerImpl(k), ctx
}
