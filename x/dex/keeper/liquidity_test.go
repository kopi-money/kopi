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

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	require.Nil(t, err)
	poolBalance := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())
	require.Equal(t, int64(1), poolBalance.AmountOf(constants.BaseCurrency).Int64())

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	require.Nil(t, err)
	num := k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 2, num)
	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(2), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(2), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 1, num)
	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(1), liq.Int64())
	liq = k.GetLiquidityByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, int64(1), liq.Int64())

	require.NoError(t, checkCache(ctx, k))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, 0, num)
	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
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

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10)
	require.NoError(t, err)

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, liq, math.NewInt(10))
	amount2 := getSpendableAmount(ctx, k, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount2.Add(math.NewInt(10)))
	require.True(t, amount2.LT(amount1))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10)
	require.NoError(t, err)

	amount3 := getSpendableAmount(ctx, k, constants.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount3)

	require.NoError(t, checkCache(ctx, k))
}

func TestLiquidity4(t *testing.T) {
	k, _, ctx := keepertest.SetupDexMsgServer(t)

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)
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

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)

		liq = k.GetLiquiditySum(innerCtx, constants.BaseCurrency)
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

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)
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

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)

		liq = k.GetLiquiditySum(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)
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

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(0), liq.Int64())
	iterator := k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 0, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)

		liq = k.GetLiquiditySum(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(10), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 1, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(10), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 1, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)

		liq = k.GetLiquiditySum(innerCtx, constants.BaseCurrency)
		require.Equal(t, int64(20), liq.Int64())
		iterator = k.LiquidityIterator(innerCtx, constants.BaseCurrency)
		require.Equal(t, 2, len(iterator.GetAll()))

		return nil
	})

	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(20), liq.Int64())
	iterator = k.LiquidityIterator(ctx, constants.BaseCurrency)
	require.Equal(t, 2, len(iterator.GetAll()))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		return fmt.Errorf("")
	})

	liq = k.GetLiquiditySum(ctx, constants.BaseCurrency)
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
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(1), idx)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		_, err := k.AddLiquidity(innerCtx, acc, constants.BaseCurrency, math.NewInt(10))
		require.NoError(t, err)
		return nil
	})

	idx, found = k.GetLiquidityEntryNextIndex(ctx)
	require.True(t, found)
	require.Equal(t, uint64(2), idx)

	require.NoError(t, checkCache(ctx, k))
}

func TestCutLiquidity1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	denomKeeper := k.DenomKeeper.(AddDenomInterface)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable1", "1ukusd", "1_000_000", "1_000_000", "10_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable2", "1ukusd", "1_000_000", "1_000_000", "1_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable2", keepertest.Alice, 1_000000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable1", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable2", 1_000000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    "stable1",
		TradeDenomReceiving: "stable2",
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Equal(t, int64(4_000000), tradeContext.CutLiquidities.Step1.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(1_000000), tradeContext.CutLiquidities.Step1.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(40_000000), tradeContext.CutLiquidities.Step1.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000000), tradeContext.CutLiquidities.Step1.VirtualOther.TruncateInt().Int64())

	require.Equal(t, int64(4_000000), tradeContext.CutLiquidities.Step2.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(1_000000), tradeContext.CutLiquidities.Step2.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(4_000000), tradeContext.CutLiquidities.Step2.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(1_000000), tradeContext.CutLiquidities.Step2.VirtualOther.TruncateInt().Int64())
}

func TestCutLiquidity2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Equal(t, int64(10_000), tradeContext.CutLiquidities.Step1.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(0), tradeContext.CutLiquidities.Step1.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(39_990_000), tradeContext.CutLiquidities.Step1.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000_000), tradeContext.CutLiquidities.Step1.VirtualOther.TruncateInt().Int64())

	require.Nil(t, tradeContext.CutLiquidities.Step2)
}

func TestCutLiquidity3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 200_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Equal(t, int64(500_000), tradeContext.CutLiquidities.Step1.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(200_000), tradeContext.CutLiquidities.Step1.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(40_000_000), tradeContext.CutLiquidities.Step1.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(9_925_000), tradeContext.CutLiquidities.Step1.VirtualOther.TruncateInt().Int64())

	require.Equal(t, int64(500_000), tradeContext.CutLiquidities.Step2.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(100_000), tradeContext.CutLiquidities.Step2.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(39_900_000), tradeContext.CutLiquidities.Step2.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000_000), tradeContext.CutLiquidities.Step2.VirtualOther.TruncateInt().Int64())
}

func TestCutLiquidity4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Equal(t, int64(50_000), tradeContext.CutLiquidities.Step1.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(5_000), tradeContext.CutLiquidities.Step1.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(39_970_000), tradeContext.CutLiquidities.Step1.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000_000), tradeContext.CutLiquidities.Step1.VirtualOther.TruncateInt().Int64())
}

func TestCutLiquidity5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Equal(t, int64(1_000_000), tradeContext.CutLiquidities.Step1.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(1_000_000), tradeContext.CutLiquidities.Step1.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(40_000_000), tradeContext.CutLiquidities.Step1.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(9_250_000), tradeContext.CutLiquidities.Step1.VirtualOther.TruncateInt().Int64())

	require.Nil(t, tradeContext.CutLiquidities.Step2)
}

func TestCutLiquidity6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	require.Nil(t, tradeContext.CutLiquidities.Step1)

	require.Equal(t, int64(1_000_000), tradeContext.CutLiquidities.Step2.CutBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000), tradeContext.CutLiquidities.Step2.CutOther.TruncateInt().Int64())
	require.Equal(t, int64(39_040_000), tradeContext.CutLiquidities.Step2.VirtualBase.TruncateInt().Int64())
	require.Equal(t, int64(10_000_000), tradeContext.CutLiquidities.Step2.VirtualOther.TruncateInt().Int64())
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
