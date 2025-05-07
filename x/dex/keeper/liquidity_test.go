package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
)

func TestLiquidity1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1))
	poolBalance := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())
	require.Equal(t, int64(1), poolBalance.AmountOf(constants.BaseCurrency).Int64())

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1))
	num := k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 2, num)
	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(2), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(2), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1))
	num = k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 1, num)
	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1))
	num = k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 0, num)
	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(0), liq.Int64())

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Carol, constants.BaseCurrency, 1)

	var idx uint64 = 0
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	for iterator.Valid() {
		liq := iterator.GetNext()

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}

	_ = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)

	idx = 0
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	for iterator.Valid() {
		liq := iterator.GetNext()

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount1 := getSpendableAmount(ctx, k, constants.BaseCurrency, keepertest.Alice)
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10))

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, liq, math.NewInt(10))
	amount2 := getSpendableAmount(ctx, k, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount2.Add(math.NewInt(10)))
	require.True(t, amount2.LT(amount1))

	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10))

	amount3 := getSpendableAmount(ctx, k, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount3)

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity4(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity5(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))

		liq = k.GetPoolLiquidity(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity6(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity7(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))

		liq = k.GetPoolLiquidity(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	//require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity8(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))

		liq = k.GetPoolLiquidity(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))

		liq = k.GetPoolLiquidity(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(20), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 2, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(20), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 2, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		return fmt.Errorf("")
	})

	liq = k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(20), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 2, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity9(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	idx, found := k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(0), idx)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(1), idx)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10)))
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(2), idx)

	require.NoError(t, checkCache(ctx, k))
}

func getSpendableAmount(ctx context.Context, k keeper.Keeper, denom, address string) math.Int {
	addr, _ := sdk.AccAddressFromBech32(address)
	coins := k.BankKeeper.SpendableCoins(ctx, addr)

	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.Int{}
}
