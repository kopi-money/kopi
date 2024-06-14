package keeper

import (
	"github.com/kopi-money/kopi/cache"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/kopi-money/kopi/utils"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	mmkeeper "github.com/kopi-money/kopi/x/mm/keeper"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
	"github.com/stretchr/testify/require"
)

func MmKeeper(t *testing.T) (dexkeeper.Keeper, mmkeeper.Keeper, sdk.Context) {
	dexKeeper, ctx, keys := DexKeeper(t)

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	mmKeeper := mmkeeper.NewKeeper(
		keys.cdc,
		runtime.NewKVStoreService(keys.mm),
		log.NewNopLogger(),
		dexKeeper.AccountKeeper,
		dexKeeper.BankKeeper,
		dexKeeper.DenomKeeper.(mmtypes.DenomKeeper),
		dexKeeper,
		authority.String(),
	)
	cache.AddCache(mmKeeper)

	require.NoError(t, mmKeeper.SetParams(ctx, MMTestingParams()))

	accountKeeper := mmKeeper.AccountKeeper.(authkeeper.AccountKeeper)

	addresses := []string{mmtypes.ModuleName, mmtypes.PoolVault, mmtypes.PoolCollateral, mmtypes.PoolRedemption}
	for _, moduleAddressName := range addresses {
		// not necesary to give all permissions to all accounts, but it's easier...
		acc := authtypes.NewEmptyModuleAccount(moduleAddressName, authtypes.Minter, authtypes.Burner)
		accI := accountKeeper.NewAccountWithAddress(ctx, sdk.AccAddress(acc.Address))
		require.NoError(t, acc.SetAccountNumber(accI.GetAccountNumber()))
		accountKeeper.SetAccount(ctx, acc)
	}

	return dexKeeper, mmKeeper, ctx
}

func MMTestingParams() mmtypes.Params {
	return mmtypes.Params{
		ProtocolShare:    math.LegacyNewDecWithPrec(5, 1),
		MinRedemptionFee: math.LegacyNewDecWithPrec(1, 1),
		MinInterestRate:  math.LegacyNewDecWithPrec(5, 2),
		A:                math.LegacyNewDec(14),
		B:                math.LegacyNewDec(131072),
	}
}

func SetupMMMsgServer(t *testing.T) (mmkeeper.Keeper, dextypes.MsgServer, mmtypes.MsgServer, sdk.Context) {
	dexK, mmK, ctx := MmKeeper(t)
	addFunds(ctx, mmK.BankKeeper.(bankkeeper.BaseKeeper), t)

	dexMsg := dexkeeper.NewMsgServerImpl(dexK)
	mmMsg := mmkeeper.NewMsgServerImpl(mmK)

	err := AddLiquidity(ctx, dexMsg, Alice, utils.BaseCurrency, Pow(2))
	require.Nil(t, err)
	err = AddLiquidity(ctx, dexMsg, Alice, "ukusd", Pow(2))
	require.Nil(t, err)

	return mmK, dexMsg, mmMsg, ctx
}
