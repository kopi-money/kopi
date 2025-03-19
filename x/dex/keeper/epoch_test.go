package keeper_test

import (
	"context"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"testing"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestEpoch1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CreateNewSnapshot(innerCtx)
	}))

	require.Equal(t, math.LegacyNewDec(1), k.GetEpochShareSum(ctx))

	epochShares := k.GetEpochSharesPerAddress(ctx, keepertest.Alice)
	require.Equal(t, 1, len(epochShares))
	require.Equal(t, int64(1), epochShares[0].Shares.TruncateInt64())

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	feeAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeIncome)
	fundsToDistribute := k.BankKeeper.SpendableCoins(ctx, feeAcc.GetAddress())
	require.Equal(t, 1, len(fundsToDistribute))
	require.Equal(t, constants.BaseCurrency, fundsToDistribute[0].Denom)
	require.Equal(t, int64(1), fundsToDistribute[0].Amount.Int64())

	epochShareSum := k.GetEpochShareSum(ctx)
	require.Equal(t, int64(1), epochShareSum.TruncateInt64())

	epochLeftovers := k.GetEpochLeftovers(ctx, keepertest.Alice, 1)
	require.Equal(t, 0, len(epochLeftovers.Leftovers))

	_, has := k.GetEpochShares(ctx, keepertest.Alice, 1)
	require.True(t, has)

	coins = sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(998)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.DistributeCollectedFees(innerCtx)
	}))

	epochLeftovers = k.GetEpochLeftovers(ctx, keepertest.Alice, 1)
	require.Equal(t, 1, len(epochLeftovers.Leftovers))
	require.Equal(t, math.LegacyNewDec(999), epochLeftovers.Leftovers[0].Amount)

	coins = sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	balance1 := k.BankKeeper.SpendableCoin(ctx, acc, constants.BaseCurrency).Amount.Int64()
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.DistributeCollectedFees(innerCtx)
	}))

	balance2 := k.BankKeeper.SpendableCoin(ctx, acc, constants.BaseCurrency).Amount.Int64()
	require.True(t, balance2 > balance1)
}

func TestEpoch2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidityCompound(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1, true))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CreateNewSnapshot(innerCtx)
	})

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	_, has := k.GetEpochShares(ctx, keepertest.Alice, 1)
	require.True(t, has)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RestartEpoch(innerCtx)
	}))

	require.Equal(t, int64(1), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 1).Int64())

	coins = sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(999)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RestartEpoch(innerCtx)
	}))

	require.Equal(t, int64(1001), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 1).Int64())

	epochPayouts := k.GetEpochLeftovers(ctx, keepertest.Alice, 1)
	require.Equal(t, 0, len(epochPayouts.Leftovers))
}

func TestEpoch3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Dave, 2)

	require.NoError(t, keepertest.AddLiquidityCompound(ctx, msg, keepertest.Dave, constants.BaseCurrency, 1, true))
	require.NoError(t, keepertest.AddLiquidityCompound(ctx, msg, keepertest.Dave, constants.BaseCurrency, 1, true))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CreateNewSnapshot(innerCtx)
	}))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	_, has := k.GetEpochShares(ctx, keepertest.Dave, 1)
	require.True(t, has)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RestartEpoch(innerCtx)
	}))

	require.Equal(t, int64(1), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 1).Int64())
	require.Equal(t, int64(1), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 2).Int64())

	var epochPayouts types.EpochLeftovers
	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 1)
	require.Equal(t, 1, len(epochPayouts.Leftovers))
	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 2)
	require.Equal(t, 1, len(epochPayouts.Leftovers))

	coins = sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1000)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RestartEpoch(innerCtx)
	}))

	require.Equal(t, int64(1), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 1).Int64())
	require.Equal(t, int64(1), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 2).Int64())

	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 1)
	require.Equal(t, 1, len(epochPayouts.Leftovers))
	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 2)
	require.Equal(t, 1, len(epochPayouts.Leftovers))

	coins = sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1000)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.RestartEpoch(innerCtx)
	}))

	require.Equal(t, int64(1001), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 1).Int64())
	require.Equal(t, int64(1001), k.GetLiquidityByPositionIndex(ctx, constants.BaseCurrency, 2).Int64())

	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 1)
	require.Equal(t, 1, len(epochPayouts.Leftovers))
	require.Equal(t, math.LegacyNewDecWithPrec(5, 1), epochPayouts.Leftovers[0].Amount)
	epochPayouts = k.GetEpochLeftovers(ctx, keepertest.Dave, 2)
	require.Equal(t, math.LegacyNewDecWithPrec(5, 1), epochPayouts.Leftovers[0].Amount)
}

func TestEpoch4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidityCompound(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1, true))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CreateNewSnapshot(innerCtx)
	}))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, math.NewInt(1)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFeeIncome, coins)

	_, has := k.GetEpochShares(ctx, keepertest.Alice, 1)
	require.True(t, has)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		addresses, err := k.GetEpochSharesAddresses(innerCtx)
		require.NoError(t, err)
		require.Equal(t, 1, len(addresses))

		if err = k.DeleteOldSnapshot(innerCtx); err != nil {
			return err
		}

		addresses, err = k.GetEpochSharesAddresses(innerCtx)
		require.NoError(t, err)
		require.Equal(t, 0, len(addresses))

		return nil
	}))
}
