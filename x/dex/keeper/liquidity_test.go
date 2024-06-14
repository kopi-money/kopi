package keeper_test

import (
	"fmt"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
)

func TestLiquidity1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	require.Nil(t, err)
	poolBalance := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())
	require.Equal(t, int64(1), poolBalance.AmountOf(utils.BaseCurrency).Int64())

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	require.Nil(t, err)
	num := k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 2, num)
	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(2), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(2), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 1, num)
	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 0, num)
	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(0), liq.Int64())

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Carol, utils.BaseCurrency, 1)

	var idx uint64 = 0
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	for iterator.Valid() {
		liq := iterator.GetNext()

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}

	_ = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)

	idx = 0
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	for iterator.Valid() {
		liq := iterator.GetNext()

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount1 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, liq, math.NewInt(10))
	amount2 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount2.Add(math.NewInt(10)))
	require.True(t, amount2.LT(amount1))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)

	amount3 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount3)

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity4(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity5(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))

		liq = k.GetLiquiditySum(innerCtx, utils.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, utils.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity6(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	//require.Equal(t, int64(0), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity7(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))

		liq = k.GetLiquiditySum(innerCtx, utils.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, utils.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))
		return fmt.Errorf("")
	})

	// Cannot be tested since the module account's balance is not rolled back
	//liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	//require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity8(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))

		liq = k.GetLiquiditySum(innerCtx, utils.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, utils.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))

		liq = k.GetLiquiditySum(innerCtx, utils.BaseCurrency)
		require.Equal(t, int64(20), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, utils.BaseCurrency)
		require.Equal(t, 2, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(20), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 2, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return fmt.Errorf("")
	})

	liq = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(20), liq.Int64())
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	require.Equal(t, 2, len(iterator.GetAll()))

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity9(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	idx, found := k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(0), idx)

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(1), idx)

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		require.NoError(t, k.AddLiquidity(innerCtx, ctx.EventManager(), acc, utils.BaseCurrency, math.NewInt(10)))
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(2), idx)

	require.NoError(t, checkCache(ctx, k))
}

func getSpendableAmount(ctx sdk.Context, k keeper.Keeper, denom, address string) math.Int {
	addr, _ := sdk.AccAddressFromBech32(address)
	coins := k.BankKeeper.SpendableCoins(ctx, addr)

	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.Int{}
}
