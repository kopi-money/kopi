package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestCreatePool1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	_, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.False(t, has)

	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "1000", "0.1", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), pool.PoolFee)
	require.Equal(t, uint64(10), pool.UnlockInSeconds)
}

func TestAddLiquidity1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "1000", "0.1", 10))

	require.NoError(t, keepertest.AddFactoryLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000"))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, math.NewInt(2000), pool.FactoryDenomAmount)
	require.Equal(t, math.NewInt(2000), pool.KCoinAmount)

	liquidityAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	balance := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(2000), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(2000), balance.AmountOf(constants.KUSD))
}

func TestAddLiquidity2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "100", "0.1", 10))

	require.NoError(t, keepertest.AddFactoryLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000"))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, math.NewInt(2000), pool.FactoryDenomAmount)
	require.Equal(t, math.NewInt(200), pool.KCoinAmount)

	liquidityAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	balance := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(2000), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(200), balance.AmountOf(constants.KUSD))
}

func TestUnlocking1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "100", "0.1", 10))

	require.NoError(t, keepertest.UnlockLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100"))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, math.NewInt(900), pool.FactoryDenomAmount)
	require.Equal(t, math.NewInt(90), pool.KCoinAmount)

	liquidityAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	balance := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(900), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(90), balance.AmountOf(constants.KUSD))

	unlockingAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolUnlocking)
	balance = k.BankKeeper.SpendableCoins(ctx, unlockingAcc.GetAddress())
	require.Equal(t, math.NewInt(100), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(10), balance.AmountOf(constants.KUSD))

	unlockings := k.GetUnlockings(ctx, factoryDenomHash, keepertest.Alice)
	require.Equal(t, 1, len(unlockings))

	require.Error(t, keepertest.UnlockLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000"))
	require.NoError(t, keepertest.UnlockLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100"))

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, math.NewInt(800), pool.FactoryDenomAmount)
	require.Equal(t, math.NewInt(80), pool.KCoinAmount)

	balance = k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(800), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(80), balance.AmountOf(constants.KUSD))

	balance = k.BankKeeper.SpendableCoins(ctx, unlockingAcc.GetAddress())
	require.Equal(t, math.NewInt(200), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(20), balance.AmountOf(constants.KUSD))

	unlockings = k.GetUnlockings(ctx, factoryDenomHash, keepertest.Alice)
	require.Equal(t, 2, len(unlockings))
}

func TestUnlocking2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "100", "0.1", 10))
	require.NoError(t, keepertest.UnlockLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100"))

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.HandleUnlockings(innerCtx, now)
		return nil
	})

	unlockings := k.GetUnlockings(ctx, factoryDenomHash, keepertest.Alice)
	require.Equal(t, 1, len(unlockings))

	now = now.Add(time.Second * 11)
	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.HandleUnlockings(innerCtx, now)
		return nil
	})

	unlockings = k.GetUnlockings(ctx, factoryDenomHash, keepertest.Alice)
	require.Equal(t, 0, len(unlockings))

	unlockingAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolUnlocking)
	balance := k.BankKeeper.SpendableCoins(ctx, unlockingAcc.GetAddress())
	require.Equal(t, math.NewInt(0), balance.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(0), balance.AmountOf(constants.KUSD))
}

func TestUpdateSettingsTest1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	_, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.False(t, has)

	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "1000", "0.1", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 1), pool.PoolFee)
	require.Equal(t, uint64(10), pool.UnlockInSeconds)

	require.NoError(t, keepertest.UpdateLiquidityPoolSettings(ctx, msgServer, keepertest.Alice, factoryDenomHash, "0.2", 100))

	pool, has = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)
	require.Equal(t, math.LegacyNewDecWithPrec(2, 1), pool.PoolFee)
	require.Equal(t, uint64(100), pool.UnlockInSeconds)
}

func TestDissolvePool1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	userAcc1 := k.BankKeeper.SpendableCoins(ctx, acc)

	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "100", "0.1", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(100), pool.KCoinAmount.Int64())

	liquidityAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	balanceModule1 := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, int64(1000), balanceModule1.AmountOf(factoryDenomHash).Int64())
	require.Equal(t, int64(100), balanceModule1.AmountOf(constants.KUSD).Int64())

	require.NoError(t, keepertest.DissolvePool(ctx, msgServer, keepertest.Alice, factoryDenomHash))

	balanceModule2 := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(0), balanceModule2.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(0), balanceModule2.AmountOf(constants.KUSD))

	userAcc2 := k.BankKeeper.SpendableCoins(ctx, acc)
	require.Equal(t, userAcc1.AmountOf(factoryDenomHash).Int64(), userAcc2.AmountOf(factoryDenomHash).Int64())
	require.Equal(t, userAcc1.AmountOf(constants.KUSD).Int64(), userAcc2.AmountOf(constants.KUSD).Int64())

	_, poolExists := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.False(t, poolExists)
}

func TestDissolvePool2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	accAlice, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	balanceAlice1 := k.BankKeeper.SpendableCoins(ctx, accAlice)

	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "100", "0.1", 10))

	shares := k.LiquidityShareIterator(ctx, factoryDenomHash).GetAll()
	require.Len(t, shares, 1)
	for _, share := range shares {
		require.Equal(t, math.LegacyOneDec(), share.Share)
	}

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(100), pool.KCoinAmount.Int64())

	liquidityAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	balanceModule1 := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, int64(1000), balanceModule1.AmountOf(factoryDenomHash).Int64())
	require.Equal(t, int64(100), balanceModule1.AmountOf(constants.KUSD).Int64())

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "1000"))

	accBob, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	balanceBob1 := k.BankKeeper.SpendableCoins(ctx, accBob)
	require.NoError(t, keepertest.AddFactoryLiquidity(ctx, msgServer, keepertest.Bob, factoryDenomHash, "1000"))

	shares = k.LiquidityShareIterator(ctx, factoryDenomHash).GetAll()
	require.Len(t, shares, 2)
	for _, share := range shares {
		require.Equal(t, math.LegacyNewDecWithPrec(5, 1), share.Share)
	}

	require.NoError(t, keepertest.DissolvePool(ctx, msgServer, keepertest.Alice, factoryDenomHash))

	balanceModule2 := k.BankKeeper.SpendableCoins(ctx, liquidityAcc.GetAddress())
	require.Equal(t, math.NewInt(0), balanceModule2.AmountOf(factoryDenomHash))
	require.Equal(t, math.NewInt(0), balanceModule2.AmountOf(constants.KUSD))

	balanceAlice2 := k.BankKeeper.SpendableCoins(ctx, accAlice)
	require.Equal(t, balanceAlice1.AmountOf(factoryDenomHash).Int64(), balanceAlice2.AmountOf(factoryDenomHash).Int64())
	require.Equal(t, balanceAlice1.AmountOf(constants.KUSD).Int64(), balanceAlice2.AmountOf(constants.KUSD).Int64())

	balanceBob2 := k.BankKeeper.SpendableCoins(ctx, accBob)
	require.Equal(t, balanceBob1.AmountOf(factoryDenomHash).Int64(), balanceBob2.AmountOf(factoryDenomHash).Int64())
	require.Equal(t, balanceBob1.AmountOf(constants.KUSD).Int64(), balanceBob2.AmountOf(constants.KUSD).Int64())
}
