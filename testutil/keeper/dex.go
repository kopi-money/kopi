package keeper

import (
	"context"
	"github.com/kopi-money/kopi/cache"
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
	cache.AddCache(dexKeeper)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		if err := dexKeeper.SetParams(innerCtx, dextypes.DefaultParams()); err != nil {
			return err
		}

		return nil
	}))

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

	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.AddLiquidity(innerCtx, &add)
		return err
	})
}

func AddOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgAddOrder) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.AddOrder(innerCtx, msg)
		return err
	})
}

func UpdateOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgUpdateOrder) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.UpdateOrder(innerCtx, msg)
		return err
	})
}

func RemoveOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgRemoveOrder) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.RemoveOrder(innerCtx, msg)
		return err
	})
}

func AddDeposit(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgAddDeposit) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.AddDeposit(innerCtx, msg)
		return err
	})
}

func AddCollateral(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgAddCollateral) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.AddCollateral(innerCtx, msg)
		return err
	})
}

func Borrow(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgBorrow) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.Borrow(innerCtx, msg)
		return err
	})
}

func RepayLoan(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgRepayLoan) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.RepayLoan(innerCtx, msg)
		return err
	})
}

func PartiallyRepayLoan(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgPartiallyRepayLoan) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.PartiallyRepayLoan(innerCtx, msg)
		return err
	})
}

func CreateRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgCreateRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.CreateRedemptionRequest(innerCtx, msg)
		return err
	})
}

func UpdateRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgUpdateRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.UpdateRedemptionRequest(innerCtx, msg)
		return err
	})
}

func CancelRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgCancelRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.CancelRedemptionRequest(innerCtx, msg)
		return err
	})
}

func Trade(ctx context.Context, k dextypes.MsgServer, msgTrade *dextypes.MsgTrade) (*dextypes.MsgTradeResponse, error) {
	var res *dextypes.MsgTradeResponse
	var err error

	err = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		res, err = k.Trade(innerCtx, msgTrade)
		return err
	})

	return res, err
}

func RemoveLiquidity(ctx context.Context, k dextypes.MsgServer, address, denom string, amount int64) error {
	rem := dextypes.MsgRemoveLiquidity{
		Creator: address,
		Denom:   denom,
		Amount:  strconv.Itoa(int(amount)),
	}

	return cache.Transact(ctx, func(innerCtx sdk.Context) error {
		_, err := k.RemoveLiquidity(innerCtx, &rem)
		return err
	})
}

func SetupDexMsgServer(t *testing.T) (dexkeeper.Keeper, dextypes.MsgServer, sdk.Context) {
	k, ctx, _ := DexKeeper(t)
	addFunds(ctx, k.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, dexkeeper.NewMsgServerImpl(k), ctx
}
