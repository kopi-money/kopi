package keeper_test

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/cache"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateSingleMaximumTradableAmount1(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(1000)
	virtualTo := math.LegacyNewDec(0)

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil, nil)
	require.Nil(t, maximum)
}

func TestCalculateSingleMaximumTradableAmount2(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(500)
	virtualTo := math.LegacyNewDec(500)

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil, nil)
	require.NotNil(t, maximum)

	receive := dexkeeper.ConstantProductTrade(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum)
	require.Equal(t, receive, actualTo)
}

func TestCalculateSingleMaximumTradableAmount3(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(100)
	virtualTo := math.LegacyNewDec(900)

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil, nil)
	require.NotNil(t, maximum)

	receive := dexkeeper.ConstantProductTrade(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum)

	// Due to rounding we don't get exactly 100, but 99.999999999999999910
	diff := actualTo.Sub(receive).Abs()
	require.True(t, diff.LT(math.LegacyNewDecWithPrec(1, 10)))
}

func TestSingleTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100))

	require.Equal(t, int64(100), k.GetLiquiditySum(ctx, utils.BaseCurrency).Int64())
	require.Equal(t, int64(100), k.GetLiquiditySum(ctx, "ukusd").Int64())
	require.Equal(t, int64(400), k.GetFullLiquidityBase(ctx, "ukusd").TruncateInt().Int64())
	require.Equal(t, int64(100), k.GetFullLiquidityOther(ctx, "ukusd").TruncateInt().Int64())

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(utils.BaseCurrency).Int64())

	fee := math.LegacyZeroDec()

	ratio1, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), ratio1.Ratio)

	pair1, _ := k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, math.LegacyNewDec(300), pair1.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair1.VirtualOther)

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(100),
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var amount math.Int
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		_, amount, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToTarget(fee, amount)
		_, _, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	liquidityPoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	liquidityPool := k.BankKeeper.SpendableCoins(ctx, liquidityPoolAcc.GetAddress())
	require.Equal(t, int64(100), liquidityPool.AmountOf(utils.BaseCurrency).Int64())
	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	ratio2, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, "0.666666666666666667", ratio2.Ratio.String())

	require.Equal(t, int64(0), k.GetLiquiditySum(ctx, utils.BaseCurrency).Int64())
	require.Equal(t, int64(200), k.GetLiquiditySum(ctx, "ukusd").Int64())
	require.Equal(t, int64(300), k.GetFullLiquidityBase(ctx, "ukusd").RoundInt().Int64())
	require.Equal(t, int64(200), k.GetFullLiquidityOther(ctx, "ukusd").TruncateInt().Int64())

	pair2, _ := k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, int64(300), pair2.VirtualBase.RoundInt().Int64())
	require.Equal(t, math.LegacyZeroDec(), pair2.VirtualOther)

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(0), coins.AmountOf(utils.BaseCurrency).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100))

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(utils.BaseCurrency).Int64())

	offer := math.NewInt(100)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amount         math.Int
		usedAmount     math.Int
		receivedAmount math.Int
		err            error
	)
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		_, amount, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToTarget(fee, amount)
		usedAmount, receivedAmount, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(200), coins.AmountOf(utils.BaseCurrency).Int64())
	require.Equal(t, int64(0), coins.AmountOf("ukusd").Int64())
	require.Equal(t, int64(100), usedAmount.Int64())
	require.Equal(t, int64(100), receivedAmount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, int64(100), liq.Int64())

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		amountUsed, amountReceived, feePaid, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceived)
		_, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		MaxPrice:         nil,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		AllowIncomplete:  false,
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		amountUsed, amountReceived, feePaid, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceived)
		_, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, math.NewInt(98), amountReceived)
	require.Equal(t, math.NewInt(2), feePaid)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		CoinSource:       keepertest.Carol,
		CoinTarget:       keepertest.Carol,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceived)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(200), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		CoinSource:       keepertest.Carol,
		CoinTarget:       keepertest.Carol,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceived)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(200), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      offer,
		CoinSource:       keepertest.Carol,
		CoinTarget:       keepertest.Carol,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceived)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(2), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115_000), pair.VirtualOther)

	ordersCache := k.NewOrdersCaches(ctx)
	maximum := k.CalculateSingleMaximumTradableAmount(ordersCache, utils.BaseCurrency, "ukusd", nil)
	require.NotNil(t, maximum)

	receivedAmount := k.ConstantProductTrade(ctx, utils.BaseCurrency, "ukusd", *maximum).RoundInt()
	liqSum := k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, receivedAmount, liqSum)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	tradeCtx := types.TradeContext{
		Context:          ctx,
		CoinSource:       keepertest.Carol,
		CoinTarget:       keepertest.Carol,
		GivenAmount:      math.NewInt(10_000),
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  false,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		amountUsed, _, amountReceived, feePaid1, feePaid2, err = k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.NoError(t, err)
	require.Equal(t, int64(2448), amountReceived.Int64())
	require.Equal(t, int64(10_000), amountUsed.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(2), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade10(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)

	tradeCtx := types.TradeContext{
		Context:          ctx,
		CoinSource:       keepertest.Carol,
		CoinTarget:       keepertest.Carol,
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: dexkeeper.FlatPrice{},
		AllowIncomplete:  false,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		amountUsed, _, amountReceived, feePaid, _, err = k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.Equal(t, int64(9_990), amountReceived.Int64())
	require.Equal(t, int64(10_000), amountUsed.Int64())
	require.Equal(t, int64(10), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))

	startAmount := int64(1000)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:         ctx,
		GivenAmount:     math.NewInt(startAmount),
		TradeDenomStart: utils.BaseCurrency,
		TradeDenomEnd:   "ukusd",
	}

	amountReceived1, _, _, err := k.SimulateTradeWithFee(tradeCtx, fee)
	require.NoError(t, err)

	tradeCtx = types.TradeContext{
		Context:         ctx,
		GivenAmount:     amountReceived1,
		TradeDenomStart: "ukusd",
		TradeDenomEnd:   utils.BaseCurrency,
	}

	amountReceived2, _, _, err := k.SimulateTradeWithFee(tradeCtx, fee)
	require.NoError(t, err)

	// Not exactly 1000 due to rounding
	require.Equal(t, int64(994), amountReceived2.Int64())
}

func TestTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2)))

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, math.NewInt(2_000_000), liq)
	liq = k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, math.NewInt(2_000_000), liq)

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, keepertest.PowDec(6), pair.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	res, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.Nil(t, err)
	require.Equal(t, int64(222000), res.AmountReceived)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)

	coins := k.BankKeeper.SpendableCoins(ctx, addr)
	coinBase := getCoin(coins, utils.BaseCurrency)
	require.Equal(t, "99999000000", coinBase.Amount.String())

	coinKUSD := getCoin(coins, "ukusd")
	expected := 100000000000 + res.AmountReceived
	require.Equal(t, expected, coinKUSD.Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(1)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(1)))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.NoError(t, err)
	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, math.NewInt(2000000), liq)
}

func TestTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, utils.BaseCurrency, keepertest.Pow(2)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(1)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", keepertest.Pow(1)))

	_, _ = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(2),
	})

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, liq, math.NewInt(6_000_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(1)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", keepertest.Pow(1)))

	_, _ = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.Equal(t, math.NewInt(2_000_000), liq)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2)))

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})
	require.NoError(t, err)

	pair, _ := k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, int64(6000000), pair.VirtualBase.RoundInt().Int64())
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2)))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2)))

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.NoError(t, err)

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	require.Equal(t, 1, len(coins))
	require.Equal(t, int64(111), coins[0].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000))

	_, _ = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "2000",
	})

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.True(t, pair.VirtualBase.GTE(math.LegacyZeroDec()))
	require.True(t, pair.VirtualOther.GTE(math.LegacyZeroDec()))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000))

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	_, err = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "2000",
	})
	require.NoError(t, err)

	price2, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000))

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	_, err = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "2000",
	})
	require.NoError(t, err)

	price2, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)
	require.True(t, price2.LT(price1))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000))

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NoError(t, err)
	require.Equal(t, 2, len(k.LiquidityIterator(ctx, "ukusd").GetAll()))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade12(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 25_000))

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NoError(t, err)

	require.Equal(t, 3, len(k.LiquidityIterator(ctx, "ukusd").GetAll()))
	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	_, err = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	response, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
		MaxPrice:  "0.01",
	})

	require.Nil(t, response)
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	response, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NotNil(t, response)
	require.NoError(t, err)

	price := float64(response.AmountUsed) / float64(response.AmountReceived)
	require.Equal(t, price, 8.00768737988469)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(11500), pair.VirtualOther)

	_, err = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "50000",
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(11500), pair.VirtualOther)

	response, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "4000",
	})

	require.NoError(t, err)
	// AmountReceived is not 1000 because of fee
	require.Equal(t, int64(924), response.AmountReceived)
	require.Equal(t, int64(4000), response.AmountUsed)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115000), pair.VirtualOther)

	response, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "10000",
	})

	require.NoError(t, err)
	// AmountReceived is not 1000 because of fee
	require.Equal(t, int64(2448), response.AmountReceived)
	require.Equal(t, int64(10000), response.AmountUsed)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade22(t *testing.T) {
	amountReceived1 := testSmallDenomTrade(t, 20_000)
	amountReceived2 := testSmallDenomTrade(t, 1000)
	require.Greater(t, amountReceived1, amountReceived2)
}

func TestTrade23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	ratio1, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)

	_, err = keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1000",
		MaxPrice:  "",
	})
	require.NoError(t, err)

	ratio2, err := k.GetRatio(ctx, "ukusd")
	require.NoError(t, err)
	require.True(t, ratio1.Ratio.GT(ratio2.Ratio))

	price2, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)
	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func testSmallDenomTrade(t *testing.T, amount int64) int64 {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 25_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount))

	response, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1000",
	})

	require.NoError(t, err)
	return response.GetAmountReceived()
}

func getCoin(coins []sdk.Coin, denom string) sdk.Coin {
	for _, coin := range coins {
		if coin.Denom == denom {
			return coin
		}
	}

	return sdk.Coin{}
}

func TestAddress(t *testing.T) {
	bz, err := sdk.GetFromBech32("axelar1txu08a5y7mylplyyvn9pwnfcderrz28eag23zj", "axelar")
	require.NoError(t, err)

	addr := sdk.AccAddress(bz)
	addrStr, err := bech32.ConvertAndEncode("migaloo", addr.Bytes())
	_ = addrStr
	require.NoError(t, err)
}

func TestTrade24(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))

	liqOtherSum1 := k.GetLiquiditySum(ctx, "ukusd")

	_, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1000",
		MaxPrice:  "",
	})
	require.NoError(t, err)

	liqOtherSum2 := k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, math.NewInt(1000), liqOtherSum2.Sub(liqOtherSum1))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade25(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount := int64(10_000)
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount))

	pair, err := k.GetLiquidityPair(ctx, "ukusd")
	require.NoError(t, err)

	A := k.GetFullLiquidityOther(ctx, "ukusd")
	B := k.GetFullLiquidityBase(ctx, "ukusd")
	b := B.Sub(pair.VirtualBase)
	c := pair.VirtualBase

	var maximum1 *math.LegacyDec
	if c.GT(math.LegacyZeroDec()) {
		m := A.Mul(b.Add(c)).Quo(c).Sub(A)
		maximum1 = &m
	}

	ordersCache := k.NewOrdersCaches(ctx)
	maximum2 := k.CalculateSingleMaximumTradableAmount(ordersCache, "ukusd", utils.BaseCurrency, nil)
	require.NotNil(t, maximum2)
	require.Equal(t, (*maximum1).TruncateInt(), (*maximum2).TruncateInt())

	A = k.GetFullLiquidityBase(ctx, "ukusd")
	B = k.GetFullLiquidityOther(ctx, "ukusd")
	b = B.Sub(pair.VirtualOther)
	c = pair.VirtualOther

	maximum1 = nil
	if c.GT(math.LegacyZeroDec()) {
		m := A.Mul(b.Add(c)).Quo(c).Sub(A)
		maximum1 = &m
	}

	//maximum2 = k.CalculateSingleMaximumTradableAmount(ctx, utils.BaseCurrency, "ukusd", nil)
	//require.Equal(t, maximum1.RoundInt(), maximum2)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade26(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount := int64(10_000)
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", amount))

	ordersCache := k.NewOrdersCaches(ctx)
	var maximum1 *math.LegacyDec
	maximum1 = k.CalculateSingleMaximumTradableAmount(ordersCache, utils.BaseCurrency, "ukusd", nil)
	maximum1 = k.CalculateSingleMaximumTradableAmount(ordersCache, "uwusdc", utils.BaseCurrency, maximum1)

	var maximum2 math.Int
	maximum2, err := k.CalculateMaximumTradableAmount(ctx, ordersCache, math.LegacyZeroDec(), "uwusdc", "ukusd")
	require.NoError(t, err)

	require.NotNil(t, maximum2)
	require.Equal(t, maximum1.TruncateInt().Int64(), maximum2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade27(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade28(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade30(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000))

	maxPrice := math.LegacyNewDecWithPrec(105, 1)
	res, err := keepertest.Trade(ctx, msg, &types.MsgTrade{
		Creator:         keepertest.Carol,
		DenomFrom:       "uwusdc",
		DenomTo:         "ukusd",
		Amount:          "1000",
		MaxPrice:        maxPrice.String(),
		AllowIncomplete: true,
	})

	require.NoError(t, err)
	require.True(t, res.AmountReceived > 0)

	maxPriceF, _ := maxPrice.Float64()

	var paidPrice float64
	if res.AmountReceived > 0 {
		paidPrice = float64(res.AmountUsed) / float64(res.AmountReceived)
	}

	require.LessOrEqual(t, paidPrice, maxPriceF)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade31(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10))

	ordersCache := k.NewOrdersCaches(ctx)
	fee := math.LegacyZeroDec()

	maximumTradableAmount, err := k.CalculateMaximumTradableAmount(ctx, ordersCache, fee, "ukusd", utils.BaseCurrency)
	require.NoError(t, err)

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      maximumTradableAmount,
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		OrdersCaches:     k.NewOrdersCaches(ctx),
		TradeBalances:    dexkeeper.NewTradeBalances(),
	}

	var amountReceivedNet math.Int

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeToBase(fee)
		_, amountReceivedNet, _, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeToTarget(fee, amountReceivedNet)
		_, _, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(1000),
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		AllowIncomplete:  true,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err := k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade33(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(1000),
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err := k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10))

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(10000),
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "uwusdc",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err := k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade35(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(1000),
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err := k.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade36(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins1 := k.BankKeeper.SpendableCoins(ctx, addr)
	funds1_uwusdc := coins1.AmountOf("uwusdc").Int64()
	funds1_ukusd := coins1.AmountOf("ukusd").Int64()

	tradeAmount := int64(1000)

	liq1_uwusdc := k.GetLiquiditySum(ctx, "uwusdc")
	liq1_ukusd := k.GetLiquiditySum(ctx, "ukusd")

	tradeCtx := types.TradeContext{
		Context:          ctx,
		GivenAmount:      math.NewInt(tradeAmount),
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		AllowIncomplete:  true,
		OrdersCaches:     k.NewOrdersCaches(ctx),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err := k.ExecuteTrade(tradeCtx)
		return err
	}))

	coins2 := k.BankKeeper.SpendableCoins(ctx, addr)
	funds2_uwusdc := coins2.AmountOf("uwusdc").Int64()
	funds2_ukusd := coins2.AmountOf("ukusd").Int64()

	liq2_uwusdc := k.GetLiquiditySum(ctx, "uwusdc")
	liq2_ukusd := k.GetLiquiditySum(ctx, "ukusd")

	require.Equal(t, funds1_uwusdc-tradeAmount, funds2_uwusdc)
	require.Equal(t, funds1_ukusd+tradeAmount, funds2_ukusd)

	require.Equal(t, liq1_uwusdc.Int64()+tradeAmount, liq2_uwusdc.Int64())
	require.Equal(t, liq1_ukusd.Int64()-tradeAmount, liq2_ukusd.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liqSum := k.GetLiquiditySum(ctx, denom).Int64()
		summedLiq := k.SumLiquidity(ctx, denom).Int64()
		funds := coins.AmountOf(denom).Int64()

		if denom == "ukopi" || denom == "ukusd" {
			fmt.Println(denom)
			fmt.Println(fmt.Sprintf("liqSum: %v", liqSum))
			fmt.Println(fmt.Sprintf("summedLiq: %v", summedLiq))
			fmt.Println(fmt.Sprintf("funds: %v", funds))
		}

		if liqSum != funds {
			return false
		}

		if summedLiq != funds {
			return false
		}
	}

	return true
}

func tradePoolEmpty(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolTrade)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, coin := range coins {
		if coin.Amount.GT(math.ZeroInt()) {
			return false
		}
	}

	return true
}

func checkCache(ctx context.Context, k dexkeeper.Keeper) error {
	return k.CheckCache(ctx)
}
