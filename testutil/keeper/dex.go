package keeper

import (
	"context"
	reservetypes "github.com/kopi-money/kopi/x/reserve/types"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	tokenfactorytypes "github.com/kopi-money/kopi/x/tokenfactory/types"

	"github.com/cosmos/cosmos-sdk/cache"

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
	strategiestypes "github.com/kopi-money/kopi/x/strategies/types"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"
)

func DexKeeper(t *testing.T) (dexkeeper.Keeper, context.Context, *Keys) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		govtypes.ModuleName:        {authtypes.Burner},
		distrtypes.ModuleName:      nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
		dextypes.PoolLiquidity:                 nil,
		dextypes.PoolTrade:                     nil,
		dextypes.PoolOrders:                    nil,
		dextypes.PoolReserve:                   nil,
		denomtypes.ModuleName:                  nil,
		minttypes.ModuleName:                   nil,
		mmtypes.PoolCollateral:                 nil,
		mmtypes.PoolVault:                      nil,
		mmtypes.PoolRedemption:                 nil,
		mmtypes.ModuleName:                     {authtypes.Minter},
		reservetypes.ModuleName:                {authtypes.Minter, authtypes.Burner},
		swaptypes.ModuleName:                   {authtypes.Minter, authtypes.Burner},
		strategiestypes.PoolArbitrage:          {authtypes.Minter, authtypes.Burner},
		strategiestypes.PoolAutomationFunds:    nil,
		tokenfactorytypes.ModuleName:           {authtypes.Burner, authtypes.Minter},
		tokenfactorytypes.PoolFactoryLiquidity: nil,
		tokenfactorytypes.PoolUnlocking:        nil,
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

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
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
	return AddLiquidityString(ctx, k, address, denom, strconv.Itoa(int(amount)))
}

func AddLiquidityString(ctx context.Context, k dextypes.MsgServer, address, denom, amount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.AddLiquidity(innerCtx, &dextypes.MsgAddLiquidity{
			Creator: address,
			Denom:   denom,
			Amount:  amount,
		})
		return err
	})
}

func AddOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgAddOrder) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.AddOrder(innerCtx, msg)
		return err
	})
}

func UpdateOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgUpdateOrder) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.UpdateOrder(innerCtx, msg)
		return err
	})
}

func RemoveOrder(ctx context.Context, k dextypes.MsgServer, msg *dextypes.MsgRemoveOrder) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.RemoveOrder(innerCtx, msg)
		return err
	})
}

func AddDeposit(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgAddDeposit) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.AddDeposit(innerCtx, msg)
		return err
	})
}

func AddCollateral(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgAddCollateral) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.AddCollateral(innerCtx, msg)
		return err
	})
}

func Borrow(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgBorrow) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.Borrow(innerCtx, msg)
		return err
	})
}

func RepayLoan(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgRepayLoan) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.RepayLoan(innerCtx, msg)
		return err
	})
}

func PartiallyRepayLoan(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgPartiallyRepayLoan) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.PartiallyRepayLoan(innerCtx, msg)
		return err
	})
}

func CreateRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgCreateRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.CreateRedemptionRequest(innerCtx, msg)
		return err
	})
}

func UpdateRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgUpdateRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.UpdateRedemptionRequest(innerCtx, msg)
		return err
	})
}

func CancelRedemptionRequest(ctx context.Context, k mmtypes.MsgServer, msg *mmtypes.MsgCancelRedemptionRequest) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.CancelRedemptionRequest(innerCtx, msg)
		return err
	})
}

func Sell(ctx context.Context, k dextypes.MsgServer, msgTrade *dextypes.MsgSell) (*dextypes.MsgTradeResponse, error) {
	var res *dextypes.MsgTradeResponse
	var err error

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.Sell(innerCtx, msgTrade)
		return err
	})

	return res, err
}

func Buy(ctx context.Context, k dextypes.MsgServer, msgTrade *dextypes.MsgBuy) (*dextypes.MsgTradeResponse, error) {
	var res *dextypes.MsgTradeResponse
	var err error

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.Buy(innerCtx, msgTrade)
		return err
	})

	return res, err
}

func RemoveLiquidity(ctx context.Context, k dextypes.MsgServer, address, denom string, amount int64) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.RemoveLiquidity(innerCtx, &dextypes.MsgRemoveLiquidity{
			Creator: address,
			Denom:   denom,
			Amount:  strconv.Itoa(int(amount)),
		})

		return err
	})
}

type LiquidityI interface {
	AddLiquidity(context.Context, sdk.AccAddress, string, math.Int) (math.Int, error)
	GetLiquiditySum(context.Context, string) math.Int
}

func TestAddLiquidity(ctx context.Context, k LiquidityI, t *testing.T, address, denom string, amount int64) {
	addr, err := sdk.AccAddressFromBech32(address)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = k.AddLiquidity(innerCtx, addr, denom, math.NewInt(amount))
		return err
	}))

	require.Equal(t, k.GetLiquiditySum(ctx, denom).Int64(), amount)
}

func SetupDexMsgServer(t *testing.T) (dexkeeper.Keeper, dextypes.MsgServer, context.Context) {
	k, ctx, _ := DexKeeper(t)
	addFunds(ctx, k.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, dexkeeper.NewMsgServerImpl(k), ctx
}
