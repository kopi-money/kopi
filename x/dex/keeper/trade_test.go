package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	"github.com/kopi-money/kopi/x/dex/constant_product"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateSingleMaximumSellableAmount1(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(1000)
	virtualTo := math.LegacyNewDec(0)

	maximum := dexkeeper.CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil)
	require.Nil(t, maximum)
}

func TestCalculateSingleMaximumSellableAmount2(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(500)
	virtualTo := math.LegacyNewDec(500)

	maximum := dexkeeper.CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil)
	require.NotNil(t, maximum)

	receive, _, _ := constant_product.ConstantProductTradeSell(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum, math.LegacyZeroDec())
	require.Equal(t, receive, actualTo)
}

func TestCalculateSingleMaximumSellableAmount3(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(100)
	virtualTo := math.LegacyNewDec(900)

	maximum := dexkeeper.CalculateSingleMaximumSellableAmount(actualFrom, actualTo, virtualFrom, virtualTo, nil)
	require.NotNil(t, maximum)

	receive, _, _ := constant_product.ConstantProductTradeSell(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum, math.LegacyZeroDec())

	// Due to rounding we don't get exactly 100, but 99.999999999999999910
	diff := actualTo.Sub(receive).Abs()
	require.True(t, diff.LT(math.LegacyNewDecWithPrec(1, 10)))
}

func TestCalculateSingleMaximumSellableAmount4(t *testing.T) {
	liqFrom := math.LegacyNewDec(4966641376348)
	liqTo, _ := math.LegacyNewDecFromStr("1187591536070.805216643324621576")

	maxPrice, _ := math.LegacyNewDecFromStr("4.18418")
	fee := math.LegacyZeroDec()

	maximumGiving, _ := constant_product.CalculateMaximumGiving(liqFrom, liqTo, maxPrice)
	receiving, _, _ := constant_product.ConstantProductTradeSell(liqFrom, liqTo, maximumGiving, fee)
	price := maximumGiving.Quo(receiving) // C

	require.True(t, maxPrice.Equal(price))
	require.Greater(t, maximumGiving.TruncateInt64(), int64(0))
}

func TestCalculateSingleMaximumBuyableAmount1(t *testing.T) {
	actualTo := math.LegacyNewDec(1000)
	virtualTo := math.LegacyNewDec(0)

	maximum := dexkeeper.CalculateSingleMaximumBuyableAmount(actualTo, virtualTo)
	require.Equal(t, int64(999), maximum.Int64())
}

func TestCalculateSingleMaximumBuyableAmount2(t *testing.T) {
	actualTo := math.LegacyNewDec(1000)
	virtualTo := math.LegacyNewDec(1000)

	maximum := dexkeeper.CalculateSingleMaximumBuyableAmount(actualTo, virtualTo)
	require.Equal(t, int64(1000), maximum.Int64())
}

func TestTradeSteps1(t *testing.T) {
	poolFrom := math.LegacyNewDec(1_000_000)
	poolBase := math.LegacyNewDec(4_000_000)
	poolTo := math.LegacyNewDec(1_000_000)
	maxPrice := math.LegacyNewDecWithPrec(101, 2)

	maxAmount, _ := constant_product.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)
	amountToGive1, _, err := constant_product.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)

	intermediate, _, _ := constant_product.ConstantProductTradeBuy(poolBase, poolTo, maxAmount, math.LegacyZeroDec())
	poolBase = poolBase.Add(intermediate)
	amountToGive2, _, _ := constant_product.ConstantProductTradeBuy(poolFrom, poolBase, intermediate, math.LegacyZeroDec())

	pricePaid := amountToGive2.Quo(maxAmount) // C

	require.True(t, maxPrice.GTE(pricePaid))
	require.Equal(t, amountToGive1.TruncateInt().Int64(), amountToGive2.TruncateInt().Int64())
}

func TestTradeSteps2(t *testing.T) {
	poolFrom := math.LegacyNewDec(1_000_000_000)
	poolBase := math.LegacyNewDec(500_000)
	poolTo := math.LegacyNewDec(1_000_000_000)
	maxPrice := math.LegacyNewDecWithPrec(101, 2)

	maxAmount, _ := constant_product.CalculateMaximumReceiving(poolFrom, poolTo, maxPrice)

	amountToGive1, _, err := constant_product.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount, math.LegacyZeroDec())
	require.NoError(t, err)

	intermediate, _, _ := constant_product.ConstantProductTradeBuy(poolBase, poolTo, maxAmount, math.LegacyZeroDec())
	poolBase = poolBase.Add(intermediate)
	amountToGive2, _, _ := constant_product.ConstantProductTradeBuy(poolFrom, poolBase, intermediate, math.LegacyZeroDec())

	pricePaid := amountToGive2.Quo(maxAmount) // C

	require.True(t, maxPrice.GTE(pricePaid))
	require.Equal(t, amountToGive1.TruncateInt().Int64(), amountToGive2.TruncateInt().Int64())
}

func TestSingleTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	require.Equal(t, int64(100), k.GetLiquiditySum(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(100), k.GetLiquiditySum(ctx, constants.KUSD).Int64())
	require.Equal(t, int64(400), k.GetFullLiquidityBase(ctx, constants.KUSD).TruncateInt().Int64())
	require.Equal(t, int64(100), k.GetFullLiquidityOther(ctx, constants.KUSD).TruncateInt().Int64())

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(constants.BaseCurrency).Int64())

	fee := math.LegacyZeroDec()

	ratio1, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), ratio1.Ratio)

	pair1, _ := k.GetLiquidityPair(ctx, constants.KUSD)
	require.Equal(t, math.LegacyNewDec(300), pair1.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair1.VirtualOther)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           &constant_product.FlatPrice{},
		Fee:                 fee,
	}

	var (
		amountUsed     math.Int
		amountReceived math.Int
	)

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.Equal(t, int64(100), amountUsed.Int64())
	require.Equal(t, int64(100), amountReceived.Int64())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountUsed, types.TradeTypeSell)
		amountUsed, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.Equal(t, int64(100), amountUsed.Int64())
	require.Equal(t, int64(100), amountReceived.Int64())
	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	liquidityPoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	liquidityPool := k.BankKeeper.SpendableCoins(ctx, liquidityPoolAcc.GetAddress())
	require.Equal(t, int64(0), liquidityPool.AmountOf(constants.BaseCurrency).Int64())

	//ratio2, err := k.GetRatio(ctx, constants.KUSD)
	//require.NoError(t, err)
	//require.Equal(t, "0.666666666666666667", ratio2.Ratio.String())

	require.Equal(t, int64(0), k.GetLiquiditySum(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(200), k.GetLiquiditySum(ctx, constants.KUSD).Int64())
	require.Equal(t, int64(800), k.GetFullLiquidityBase(ctx, constants.KUSD).RoundInt().Int64())
	require.Equal(t, int64(200), k.GetFullLiquidityOther(ctx, constants.KUSD).TruncateInt().Int64())

	pair2, _ := k.GetLiquidityPair(ctx, constants.KUSD)
	require.Equal(t, int64(800), pair2.VirtualBase.RoundInt().Int64())
	require.Equal(t, math.LegacyZeroDec(), pair2.VirtualOther)

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(0), coins.AmountOf(constants.BaseCurrency).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade1a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           &constant_product.FlatPrice{},
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeBuy)
		amountUsed, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.Equal(t, int64(100), amountUsed.Int64())
	require.Equal(t, int64(100), amountReceived.Int64())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountUsed, types.TradeTypeBuy)
		amountUsed, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(100), amountUsed.Int64())

	require.Equal(t, int64(0), k.GetLiquiditySum(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(200), k.GetLiquiditySum(ctx, constants.KUSD).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(constants.BaseCurrency).Int64())

	offer := math.NewInt(100)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           &constant_product.FlatPrice{},
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amount         math.Int
		usedAmount     math.Int
		receivedAmount math.Int
		err            error
	)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		_, amount, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amount, types.TradeTypeSell)
		usedAmount, receivedAmount, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), usedAmount.Int64())
	require.Equal(t, int64(100), receivedAmount.Int64())
	require.Equal(t, int64(200), coins.AmountOf(constants.BaseCurrency).Int64())
	require.Equal(t, int64(0), coins.AmountOf(constants.KUSD).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(100), liq.Int64())

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		FlatPrice:           &constant_product.FlatPrice{},
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, feePaid, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeSell)
		_, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade3a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, int64(100), liq.Int64())

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		FlatPrice:           &constant_product.FlatPrice{},
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeBuy)
		amountUsed, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeBuy)
		amountUsed, amountReceived, feePaid, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(102), amountUsed.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 50))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 50))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		MaxPrice:            nil,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		MinimumTradeAmount:  &offer,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid        math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, feePaid, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeSell)
		_, amountReceived, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		MinimumTradeAmount:  &offer,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeSell)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(200), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		MinimumTradeAmount:  &offer,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeSell)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(200), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 50))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 50))

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         offer,
		MinimumTradeAmount:  &offer,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 fee,
	}

	require.NoError(t, k.PrepareCutLiquidity(&tradeCtx))

	var (
		amountUsed     math.Int
		amountReceived math.Int
		feePaid1       math.Int
		feePaid2       math.Int
		err            error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		amountUsed, amountReceived, feePaid1, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceived, types.TradeTypeSell)
		_, amountReceived, feePaid2, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(0), feePaid1.Int64())
	require.Equal(t, int64(2), feePaid2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeAmount:         offer,
		MinimumTradeAmount:  &offer,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 2),
	}

	var tradeResult types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.NoError(t, err)
	require.Equal(t, int64(2_487), tradeResult.AmountReceived.Int64())
	require.Equal(t, int64(10_000), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(12), tradeResult.FeeBase.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade10(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeAmount:         offer,
		MinimumTradeAmount:  &offer,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(2, 3),
	}

	var tradeResult types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(9_990), tradeResult.AmountReceived.Int64())
	require.Equal(t, int64(10_000), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(10), tradeResult.FeeOther.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	startAmount := int64(1000)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(startAmount),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
	}

	tradeResult1, err := k.SimulateSellWithFee(tradeCtx, fee)
	require.NoError(t, err)

	tradeCtx = types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeResult1.AmountReceived,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
	}

	tradeResult2, err := k.SimulateSellWithFee(tradeCtx, fee)
	require.NoError(t, err)

	// Not exactly 1000 due to rounding
	require.Equal(t, int64(995), tradeResult2.AmountReceived.Int64())
}

func TestSingleTrade11a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	r, _ := math.LegacyNewDecFromStr("0.000077873443626943")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	startAmount := int64(1000)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(startAmount),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
	}

	_, err := k.SimulateSellWithFee(tradeCtx, fee)
	require.NoError(t, err)
}

func TestSingleTrade12(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 5_151_753_574_086)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 8_968_155_459_761)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 98_236_914_564)

	r1, _ := math.LegacyNewDecFromStr("0.246159291770911093")
	r2, _ := math.LegacyNewDecFromStr("0.246159273456957244")

	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r2)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_151_753_574_086))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 8_968_155_459_761))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 98_236_914_564))

	maxPrice, err := math.LegacyNewDecFromStr("1.001502754382700116")
	require.NoError(t, err)

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(8_985_450_836),
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: "uwusdc",
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
		})

		return err
	}))

	pricePaid, err := res.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GT(pricePaid))
}

func TestSingleTrade13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	pricePaid, err := res.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))
}

func TestSingleTrade14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	maxPrice, err := math.LegacyNewDecFromStr("4.4448")
	require.NoError(t, err)

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(2_500),
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
		})

		return err
	}))

	pricePaid, err := res.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))
}

func TestSingleTrade15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	maxPrice, err := math.LegacyNewDecFromStr("0.275141009767505847")
	require.NoError(t, err)

	zeroInt := math.ZeroInt()

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(10_000),
			MinimumTradeAmount:  &zeroInt,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: constants.BaseCurrency,
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
		})

		return err
	}))

	pricePaid, err := res.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))
}

func TestSingleTrade16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	maxPrice, err := math.LegacyNewDecFromStr("4.4448")
	require.NoError(t, err)

	zeroInt := math.ZeroInt()

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(2_500),
			MinimumTradeAmount:  &zeroInt,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
		})

		return err
	}))

	pricePaid, err := res.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))
}

func TestSingleTrade17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	maxPrice, err := math.LegacyNewDecFromStr("1.025504291735460913")
	require.NoError(t, err)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		MaxPrice:            &maxPrice,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid1))

	zeroInt := math.ZeroInt()

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MinimumTradeAmount = &zeroInt
		res2, err = k.ExecuteSell(tradeContext)
		return err
	}))

	pricePaid2, err := res2.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid2))
}

func TestSingleTrade18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	maxPrice, err := math.LegacyNewDecFromStr("1.025504291735460913")
	require.NoError(t, err)

	zeroInt := math.ZeroInt()

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(100_000),
			MinimumTradeAmount:  &zeroInt,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: "uwusdc",
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
		})

		return err
	}))

	amountGiven, _ := strconv.ParseFloat(res.AmountGiven.String(), 64)
	amountReceived, _ := strconv.ParseFloat(res.AmountReceived.String(), 64)
	pricePaid := (amountGiven - 1) / (amountReceived + 1)
	mp, _ := maxPrice.Float64()

	require.True(t, pricePaid <= mp)
}

func TestSingleTrade19(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 5_151_753_574_086)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 8_968_155_459_761)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 98_236_914_564)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000_000))

	r1, _ := math.LegacyNewDecFromStr("0.25")
	r2, _ := math.LegacyNewDecFromStr("0.25")

	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r2)

	maxPrice, err := math.LegacyNewDecFromStr("1.01")
	require.NoError(t, err)

	zeroInt := math.ZeroInt()

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(9_000_000),
			MinimumTradeAmount:  &zeroInt,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: "uwusdc",
			MaxPrice:            &maxPrice,
			TradeBalances:       dexkeeper.NewTradeBalances(),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
		})

		return err
	}))

	amountGiven, _ := strconv.ParseFloat(res.AmountGiven.String(), 64)
	amountReceived, _ := strconv.ParseFloat(res.AmountReceived.String(), 64)
	pricePaid := (amountGiven - 1) / (amountReceived + 1)
	mp, _ := maxPrice.Float64()

	require.True(t, mp >= pricePaid)
}

func TestSingleTrade20(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 5_151_753_574_086)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 8_968_155_459_761)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 98_236_914_564)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000_000))

	r1, _ := math.LegacyNewDecFromStr("0.25")
	r2, _ := math.LegacyNewDecFromStr("0.25")

	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r2)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(9_000_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	zeroInt := math.ZeroInt()
	maxPrice, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MinimumTradeAmount = &zeroInt
		tradeContext.MaxPrice = &maxPrice
		res2, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	pricePaid, err := res2.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))
}

func TestSingleTrade21(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r1, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)

	tradeAmount := math.NewInt(10_000_000)

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         tradeAmount,
			MinimumTradeAmount:  &tradeAmount,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))
}

func TestSingleTrade22(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r1, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)

	tradeAmount := math.NewInt(999_960)

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         tradeAmount,
			MinimumTradeAmount:  &tradeAmount,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))

	// numbers are not exactly equal due to rounding
	require.Equal(t, tradeAmount.Int64(), res.AmountReceived.Int64())
}

func TestSingleTrade23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 40_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r1, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)

	tradeAmount := math.NewInt(10_000)

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		res, err = k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         tradeAmount,
			MinimumTradeAmount:  &tradeAmount,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))

	// numbers are not exactly equal due to rounding
	require.Equal(t, tradeAmount.Int64(), res.AmountReceived.Int64())
}

func TestSingleTrade24(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(1_000_000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         tradeAmount,
			TradeDenomGiving:    "uwusdc",
			TradeDenomReceiving: constants.KUSD,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))
}

func TestSingleTrade25(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(1_000_000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         tradeAmount,
			TradeDenomGiving:    "uwusdc",
			TradeDenomReceiving: constants.KUSD,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))
}

func TestSingleTrade26(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		res2, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade27(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade28(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade29(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(10_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade30(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(600_000),
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: constants.BaseCurrency,
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			Fee:                 math.LegacyZeroDec(),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			OrdersCaches:        k.NewOrdersCaches(innerCtx),
		})
		return err
	}))
}

func TestSingleTrade31(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		res2, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade33(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 4_000_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestSingleTrade34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(10_000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &pricePaid
		tradeContext.OrdersCaches = k.NewOrdersCaches(ctx)
		res2, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven.Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived.Int64())
}

func TestTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, math.NewInt(2_000_000), liq)
	liq = k.GetLiquiditySum(ctx, constants.KUSD)
	require.Equal(t, math.NewInt(2_000_000), liq)

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, int64(6_000000), pair.VirtualBase.TruncateInt64())
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})

	require.NoError(t, err)
	amountReceived, _ := strconv.Atoi(res.AmountReceived)
	require.Equal(t, 249_874, amountReceived)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)

	coins := k.BankKeeper.SpendableCoins(ctx, addr)
	coinBase := getCoin(coins, constants.BaseCurrency)
	require.Equal(t, "99999000000", coinBase.Amount.String())

	coinKUSD := getCoin(coins, constants.KUSD)
	expected := 100000000000 + amountReceived
	require.Equal(t, int64(expected), coinKUSD.Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})

	require.NoError(t, err)
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, math.NewInt(2000000), liq)
}

func TestTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 1_000000))

	_, _ = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "2_000000",
	})

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, liq, math.NewInt(6_000_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
}

func TestTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 1_000000))

	_, _ = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})

	liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
	require.Equal(t, math.NewInt(2_000_000), liq)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})
	require.NoError(t, err)

	require.Equal(t, "1000000", res.AmountGiven)
	require.Equal(t, "249874", res.AmountReceived)

	pair, _ := k.GetLiquidityPair(ctx, constants.KUSD)
	require.Equal(t, int64(0), pair.VirtualOther.TruncateInt64())
	require.Equal(t, int64(4_000259), pair.VirtualBase.RoundInt().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})

	require.NoError(t, err)

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	require.Equal(t, 1, len(coins))
	require.Equal(t, int64(62), coins[0].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 50000))

	_, _ = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "2000",
	})

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.True(t, pair.VirtualBase.GTE(math.LegacyZeroDec()))
	require.True(t, pair.VirtualOther.GTE(math.LegacyZeroDec()))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 50_000))

	price1, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "2_000_000",
	})
	require.NoError(t, err)

	price2, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 50000))

	price1, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "2000",
	})
	require.NoError(t, err)

	price2, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)
	require.True(t, price2.LT(price1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
	})

	require.NoError(t, err)
	require.Equal(t, 2, len(k.LiquidityIterator(ctx, constants.KUSD).GetAll()))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade12(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 25_000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
	})

	require.NoError(t, err)

	require.Equal(t, 3, len(k.LiquidityIterator(ctx, constants.KUSD).GetAll()))
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
	})
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
		MaxPrice:       "0.01",
	})

	require.Nil(t, response)
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade14a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	response, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
		MaxPrice:       "0.01",
	})

	require.Nil(t, response)
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
	})

	require.NoError(t, err)
	require.NotNil(t, response)

	amountReceived, _ := strconv.Atoi(response.AmountReceived)
	amountGiven, _ := strconv.Atoi(response.AmountGiven)

	price := float64(amountGiven) / float64(amountReceived)
	require.Equal(t, 4.002081082162724, price)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(11500), pair.VirtualOther)

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "4000",
	})

	require.NoError(t, err)
	require.Equal(t, "4000", response.AmountGiven)
	require.Equal(t, "999", response.AmountReceived)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	pair, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(115000), pair.VirtualOther)

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
	})

	require.NoError(t, err)
	require.Equal(t, "10000", response.AmountGiven)
	require.Equal(t, "2498", response.AmountReceived)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	price1, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	ratio1, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000_000",
		MaxPrice:       "",
	})
	require.NoError(t, err)

	ratio2, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)
	require.True(t, ratio1.Ratio.GT(ratio2.Ratio))

	price2, err := k.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)
	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func testSmallDenomTrade(t *testing.T, amount int64) int64 {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 25_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, amount))

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1000",
	})

	require.NoError(t, err)

	amountReceived, _ := strconv.Atoi(response.AmountReceived)
	return int64(amountReceived)
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

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))

	liqOtherSum1 := k.GetLiquiditySum(ctx, constants.KUSD)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "",
	})
	require.NoError(t, err)

	liqOtherSum2 := k.GetLiquiditySum(ctx, constants.KUSD)
	require.Equal(t, math.NewInt(1000), liqOtherSum2.Sub(liqOtherSum1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade26(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount := int64(10_000)
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", amount))

	tradeContext := types.TradeContext{
		Context:             ctx,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
	}

	k.PrepareCutLiquidity(&tradeContext)

	var maximum1 *math.LegacyDec
	maximum1 = k.CalculateSingleSellableAmount(tradeContext.CutLiquidity, constants.BaseCurrency, constants.KUSD, nil)
	maximum1 = k.CalculateSingleSellableAmount(tradeContext.CutLiquidity, "uwusdc", constants.BaseCurrency, maximum1)
	require.NotNil(t, maximum1)

	tradeContext = types.TradeContext{
		Context:             ctx,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}
	k.PrepareCutLiquidity(&tradeContext)

	var maximum2 *math.Int
	maximum2, _ = k.CalculateMaximumSellableAmount(tradeContext)

	require.NotNil(t, maximum2)
	require.Equal(t, maximum1.TruncateInt().Int64(), maximum2.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade27(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade28(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade30(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000))

	maxPrice := math.LegacyNewDecWithPrec(105, 1)
	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    "uwusdc",
		DenomReceiving: constants.KUSD,
		Amount:         "1000",
		MaxPrice:       maxPrice.String(),
	})

	require.NoError(t, err)

	amountReceived, _ := strconv.Atoi(res.AmountReceived)
	amountGiven, _ := strconv.Atoi(res.AmountGiven)

	require.True(t, amountReceived > 0)

	maxPriceF, _ := maxPrice.Float64()

	var paidPrice float64
	if amountReceived > 0 {
		paidPrice = float64(amountGiven) / float64(amountReceived)
	}

	require.LessOrEqual(t, paidPrice, maxPriceF)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade31(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}

	k.PrepareCutLiquidity(&tradeContext)

	maximumTradableAmount, _ := k.CalculateMaximumSellableAmount(tradeContext)

	require.NotNil(t, maximumTradableAmount)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         *maximumTradableAmount,
		MinimumTradeAmount:  maximumTradableAmount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	k.PrepareCutLiquidity(&tradeCtx)

	var (
		amountReceivedNet math.Int
		err               error
	)

	tradeAmount, err := k.CalculateMaximumSellableAmount(tradeCtx)
	require.NoError(t, err)
	require.NotNil(t, tradeAmount)

	tradeCtx.TradeAmount = *tradeAmount

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		stepCtx := tradeCtx.TradeStep1(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), types.TradeTypeSell)
		_, amountReceivedNet, _, err = k.ExecuteTradeStep(stepCtx)
		if err != nil {
			return err
		}

		stepCtx = tradeCtx.TradeStep2(tradeCtx.OrdersCaches.ReserveFeeShare.Get(), amountReceivedNet, types.TradeTypeSell)
		_, _, _, err = k.ExecuteTradeStep(stepCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(4, 2),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade33(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.BaseCurrency,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(4, 2),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(4, 2),
	}

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))
}

func TestTrade35(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(4, 2),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade36(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	coins1 := k.BankKeeper.SpendableCoins(ctx, addr)
	funds1_uwusdc := coins1.AmountOf("uwusdc").Int64()
	funds1_ukusd := coins1.AmountOf(constants.KUSD).Int64()

	tradeAmount := int64(1000)

	liq1_uwusdc := k.GetLiquiditySum(ctx, "uwusdc")
	liq1_ukusd := k.GetLiquiditySum(ctx, constants.KUSD)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(tradeAmount),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	coins2 := k.BankKeeper.SpendableCoins(ctx, addr)
	funds2_uwusdc := coins2.AmountOf("uwusdc").Int64()
	funds2_ukusd := coins2.AmountOf(constants.KUSD).Int64()

	liq2_uwusdc := k.GetLiquiditySum(ctx, "uwusdc")
	liq2_ukusd := k.GetLiquiditySum(ctx, constants.KUSD)

	require.Equal(t, funds1_uwusdc-tradeAmount, funds2_uwusdc)
	require.Equal(t, funds1_ukusd+tradeAmount, funds2_ukusd)

	require.Equal(t, liq1_uwusdc.Int64()+tradeAmount, liq2_uwusdc.Int64())
	require.Equal(t, liq1_ukusd.Int64()-tradeAmount, liq2_ukusd.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade37(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	liqBase := k.LiquidityIterator(ctx, constants.BaseCurrency).GetAll()
	require.Equal(t, 1, len(liqBase))
	require.Equal(t, int64(10_000), liqBase[0].Amount.Int64())

	liqOther := k.LiquidityIterator(ctx, constants.KUSD).GetAll()
	require.Equal(t, 1, len(liqOther))

	tradeAmount := int64(1000)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(tradeAmount),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           &constant_product.FlatPrice{},
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 2),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	liqBase = k.LiquidityIterator(ctx, constants.BaseCurrency).GetAll()
	require.Equal(t, 2, len(liqBase))
	require.Equal(t, int64(10_000), liqBase[0].Amount.Int64())
	require.Equal(t, int64(1_000), liqBase[1].Amount.Int64())

	liqOther = k.LiquidityIterator(ctx, constants.KUSD).GetAll()
	require.Equal(t, 2, len(liqOther))
	require.Equal(t, int64(9_000), liqOther[0].Amount.Int64())
	require.Equal(t, int64(3), liqOther[1].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade38(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	}
	k.PrepareCutLiquidity(&tradeContext)

	offer := math.LegacyNewDec(10_000)
	amountReceived, _, err := dexkeeper.CalculateSingleSell(constants.BaseCurrency, constants.KUSD, offer, math.LegacyZeroDec(), tradeContext.CutLiquidity)
	require.NoError(t, err)

	amountToGive, _, err := dexkeeper.CalculateSingleBuy(constants.BaseCurrency, constants.KUSD, amountReceived, math.LegacyZeroDec(), tradeContext.CutLiquidity)
	require.NoError(t, err)

	require.Equal(t, offer.RoundInt64(), amountToGive.RoundInt64())
}

func TestTrade39(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	res1, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_000000_000000",
	})
	require.NoError(t, err)

	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_000000_0000000",
		MaxPrice:       fmt.Sprintf("%.8f", price1),
	})
	require.Error(t, err)
}

func TestTrade40(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	res1, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "1_000000_000000",
	})
	require.NoError(t, err)

	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "1_000000_000000",
		MaxPrice:       fmt.Sprintf("%.8f", price1),
	})
	require.Error(t, err)
}

func TestTrade41(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	acc1, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	acc2, _ := sdk.AccAddressFromBech32(keepertest.Dave)

	amount := int64(10_000)
	coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, math.NewInt(amount)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc1, types.PoolReserve, coins)
	_ = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolReserve, acc2, coins)

	res, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_000",
	})
	require.NoError(t, err)

	amountGiven, _ := strconv.Atoi(res.AmountGiven)
	require.Less(t, amountGiven, 10_000)
}

func TestTrade42(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	pool := map[string]int64{
		"skbtc":   8496,
		"swbtc":   737567446,
		"ucwusdc": 4118700571,
		"ucwusdt": 492046007,
		"ukopi":   5223913832979,
		"ukusd":   479921845,
		"uwusdc":  38585199076,
		"uwusdt":  698434,
	}

	ratios := map[string]string{
		"sckbtc":     "0.000405912156352312",
		"skbtc":      "0.000896829507787797",
		"swbtc":      "0.000896829508394457",
		"uarbstusdc": "0.101556613612585061",
		"uarbstusdt": "0.101756063584109115",
		"uckusd":     "0.101478039088050841",
		"ucwusdc":    "0.249106333509990319",
		"ucwusdt":    "0.203250771531685959",
		"ukusd":      "0.196939223849083103",
		"uwusdc":     "0.211427997614305922",
		"uwusdt":     "0.186917656024168530",
	}

	balance := map[string]int64{
		"ucwusdc": 9180625,
		"ucwusdt": 3216676466,
		"ukusd":   3899152227,
	}

	for denom, amount := range balance {
		keepertest.AddFunds(ctx, t, k.BankKeeper, denom, keepertest.Dave, amount)
	}

	keepertest.SetLiquidity(ctx, k.BankKeeper, k, t, pool)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		for denom, ratio := range ratios {
			r, _ := math.LegacyNewDecFromStr(ratio)
			k.DenomKeeper.SetRatio(innerCtx, denomtypes.Ratio{Denom: denom, Ratio: r})
		}

		return nil
	}))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	acc1, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	acc2, _ := sdk.AccAddressFromBech32(keepertest.Dave)

	amount := int64(10_000)
	coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, math.NewInt(amount)))
	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc1, types.PoolReserve, coins)
	_ = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolReserve, acc2, coins)

	_, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Dave,
		DenomGiving:    "ucwusdc",
		DenomReceiving: constants.KUSD,
		Amount:         "10000000",
	})
	require.NoError(t, err)
}

func TestTrade43(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	pool := map[string]int64{
		"inj":   829132866175164798,
		"ukusd": 1740434604,
		"ukopi": 49994130281493,
	}

	ratios := map[string]string{
		"inj":   "11310893732.791635615102371449",
		"skbtc": "0.235974469935270325",
	}

	balance := map[string]int64{
		"inj": 37582987632368903,
	}

	for denom, amount := range balance {
		keepertest.AddFunds(ctx, t, k.BankKeeper, denom, keepertest.Dave, amount)
	}

	keepertest.SetLiquidity(ctx, k.BankKeeper, k, t, pool)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		for denom, ratio := range ratios {
			r, _ := math.LegacyNewDecFromStr(ratio)
			k.DenomKeeper.SetRatio(innerCtx, denomtypes.Ratio{Denom: denom, Ratio: r})
		}

		return nil
	}))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Dave,
		DenomGiving:        "inj",
		DenomReceiving:     "ukusd",
		Amount:             "1000000000000000",
		MaxPrice:           "48057996063.181141677468913534",
		MinimumTradeAmount: "1000000000000000",
	})
	require.NoError(t, err)
}

func TestTrade44(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Alice,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "uwusdc",
		Amount:             "10_000_000",
		MinimumTradeAmount: "0",
	})
	require.NoError(t, err)
}

func TestTrade45(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	acc1, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	acc2, _ := sdk.AccAddressFromBech32(keepertest.Dave)
	accLiq := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)

	amount := int64(1_000_000)
	coins := sdk.NewCoins(
		sdk.NewCoin(constants.BaseCurrency, math.NewInt(amount)),
		sdk.NewCoin(constants.KUSD, math.NewInt(amount)),
		sdk.NewCoin("uwusdc", math.NewInt(amount)),
	)

	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc1, types.PoolReserve, coins)
	_ = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolReserve, acc2, coins)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, constants.BaseCurrency, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, constants.KUSD, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, "uwusdc", amount))

	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Dave, constants.KUSD, amount))

	ratioKUSD, _ := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	ratioUSDC, _ := k.DenomKeeper.GetRatio(ctx, "uwusdc")

	require.True(t, ratioKUSD.Ratio.Equal(ratioUSDC.Ratio))

	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Dave,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "uwusdc",
		Amount:             strconv.Itoa(int(amount)),
		MinimumTradeAmount: "0",
	})
	require.NoError(t, err)

	amountGiven, _ := strconv.Atoi(res.AmountGiven)
	amountReceived, _ := strconv.Atoi(res.AmountReceived)
	pricePaid := float64(amountGiven) / float64(amountReceived)

	require.True(t, pricePaid > 1)
	require.Equal(t, 250_000, amountGiven)
	require.Equal(t, 249_874, amountReceived)

	require.Equal(t, int64(250_000), k.GetLiquidityByAddress(ctx, constants.KUSD, keepertest.Dave).Int64())
	require.Equal(t, int64(750_095), k.GetLiquidityByAddress(ctx, "uwusdc", keepertest.Dave).Int64())

	require.Equal(t, int64(250_000), k.BankKeeper.SpendableCoin(ctx, accLiq.GetAddress(), constants.KUSD).Amount.Int64())
	require.Equal(t, int64(750_095), k.BankKeeper.SpendableCoin(ctx, accLiq.GetAddress(), "uwusdc").Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
}

func TestTrade46(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("10000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 3140427631280))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1326651942499))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 23258054))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 1))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 12622692067))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5537967689))

	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "9999999999999000000"))

	rInj, _ := math.LegacyNewDecFromStr("16621294955.312039072718541316")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", rInj)

	maxPrice := "49492227072.515466692151864127"
	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Alice,
		DenomGiving:    "inj",
		DenomReceiving: constants.BaseCurrency,
		Amount:         "20210000",
		MaxPrice:       maxPrice,
	})

	require.Error(t, err)
}

func TestTrade47(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 3140427631280))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1326651942499))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 23258054))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 1))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 12622692067))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5537967689))

	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "9999999999999000000"))

	_, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:            keepertest.Alice,
		DenomGiving:        "inj",
		DenomReceiving:     constants.KUSD,
		Amount:             "20210000",
		MaxPrice:           "49492227072.515466692151864127",
		MinimumTradeAmount: "0",
	})

	require.NoError(t, err)
}

func TestTrade48(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4467855392339))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 63286126324))
	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "10002952303552758024"))

	r, _ := math.LegacyNewDecFromStr("0.335987740910407578")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	r, _ = math.LegacyNewDecFromStr("16534992150.086859516468639030")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", r)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Alice,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "inj",
		Amount:             "20000000",
		MinimumTradeAmount: "0",
	})

	require.NoError(t, err)
}

func TestTrade49(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4467855392339))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 63286126324))
	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "10002952303552758024"))

	r, _ := math.LegacyNewDecFromStr("0.335987740910407578")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	r, _ = math.LegacyNewDecFromStr("16534992150.086859516468639030")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", r)

	_, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:            keepertest.Alice,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "inj",
		Amount:             "1000000000000000000",
		MinimumTradeAmount: "0",
	})

	require.NoError(t, err)
}

func TestTrade50(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:            math.NewInt(100_000),
		MaximumAvailableAmount: math.NewInt(10_000),
		CoinSource:             keepertest.Bob,
		CoinTarget:             keepertest.Bob,
		TradeDenomGiving:       constants.BaseCurrency,
		TradeDenomReceiving:    constants.KUSD,
		OrdersCaches:           k.NewOrdersCaches(ctx),
		TradeBalances:          dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(9_985), res.AmountGiven.Int64())
}

func TestTrade51(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(10_000), res.AmountReceived.Int64())
}

func TestTrade52(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(10_000), res.AmountReceived.Int64())
}

func TestTrade53(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(10_000), res.AmountReceived.Int64())
}

func TestTrade54(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(9_994), res.AmountReceived.Int64())
}

func TestTrade55(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(9_994), res.AmountReceived.Int64())
}

func TestTrade56(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(9_994), res.AmountReceived.Int64())
}

func TestTrade57(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(2_499), res.AmountReceived.Int64())
}

func TestTrade58(t *testing.T) {
	price1 := scenario(t, 10_000_000_000000)
	price2 := scenario(t, 1_000_000_000_000000)
	require.Equal(t, price1, price2)
}

func scenario(t *testing.T, baseLiq int64) math.LegacyDec {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, baseLiq)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, baseLiq))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1000_000000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	price, err := res.PricePaid()
	require.NoError(t, err)
	return price
}

func TestTrade59(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(10_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		_, err := k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
	require.True(t, liquidityBalanced(ctx, k))
}

func TestTrade60(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	maxPrice, _ := math.LegacyNewDecFromStr("1.11")

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(10_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		MaxPrice:            &maxPrice,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		_, err := k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
	require.True(t, liquidityBalanced(ctx, k))
}

func TestTrade61(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 49_967842_769650)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "inj", keepertest.Alice, 1000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "inj", keepertest.Bob, 1000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 49_967842_769650))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 9958_763714))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "inj", 76706))

	r1, _ := math.LegacyNewDecFromStr("0.228660473988140735")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)

	r2, _ := math.LegacyNewDecFromStr("0.344058910461165912")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", r2)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "inj",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		_, err := k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade62(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 40_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 1000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 40_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	var res1 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		res1, err = k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(100_000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    "uwusdc",
			TradeDenomReceiving: constants.KUSD,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
		})

		return err
	}))

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res2 types.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		var err error
		res2, err = k.ExecuteSell(tradeContext)

		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	price1, _ := res1.PricePaid()
	price2, _ := res2.PricePaid()

	require.True(t, res2.AmountReceived.LT(res1.AmountReceived))
	require.True(t, price1.LT(price2))
}

func TestTrade63(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeBalances := dexkeeper.NewTradeBalances()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(5_000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: "uwusdc",
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       tradeBalances,
		})
		return err
	}))

	require.NoError(t, tradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade64(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 49967842769650)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Bob, 10_000_000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 49967842769650))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 76706))

	r, _ := math.LegacyNewDecFromStr("0.228660473988140735")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	for range 10 {
		tradeBalances := dexkeeper.NewTradeBalances()
		var res types.TradeResult
		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			var err error
			res, err = k.ExecuteSell(types.TradeContext{
				Context:             innerCtx,
				TradeAmount:         math.NewInt(10_000_000000),
				CoinSource:          keepertest.Bob,
				CoinTarget:          keepertest.Bob,
				TradeDenomGiving:    constants.KUSD,
				TradeDenomReceiving: constants.BaseCurrency,
				OrdersCaches:        k.NewOrdersCaches(ctx),
				TradeBalances:       tradeBalances,
			})
			return err
		}))

		_ = res
		require.NoError(t, tradeBalances.Settle(ctx, k.BankKeeper))
	}
}

func TestTrade65(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 49976153908117)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 1000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 49976153908117))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteSell(tradeContext)

		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade66(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade67(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade68(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade69(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade70(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyNewDecWithPrec(1, 3),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		tradeContext.MaxPrice = &maxPrice
		_, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade71(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	maxPrice, _ := math.LegacyNewDecFromStr("0.1")

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(100_000_000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: constants.BaseCurrency,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			MaxPrice:            &maxPrice,
			Fee:                 math.LegacyZeroDec(),
		})
		return err
	}))
}

func TestTrade72(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 200_000_000_000)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(1_000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    "uwusdc",
			TradeDenomReceiving: constants.KUSD,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			Fee:                 math.LegacyZeroDec(),
		})
		return err
	}), "trade amount too small")
}

func TestTrade73(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 200_000_000_000)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteSell(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(1_000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			Fee:                 math.LegacyZeroDec(),
		})
		return err
	}), "trade amount too small")
}

func TestTrade74(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 200_000_000_000)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(1000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    "uwusdc",
			TradeDenomReceiving: constants.KUSD,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			Fee:                 math.LegacyZeroDec(),
		})
		return err
	}), "trade amount too small")
}

func TestTrade75(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 200_000_000_000)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteBuy(types.TradeContext{
			Context:             innerCtx,
			TradeAmount:         math.NewInt(1000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    constants.BaseCurrency,
			TradeDenomReceiving: constants.KUSD,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			Fee:                 math.LegacyZeroDec(),
		})
		return err
	}), "trade amount too small")
}

func TestTrade76(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50037598073771)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 4925)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 68529039897008)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1000_005000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	denomKeeper, ok := k.DenomKeeper.(keepertest.SetMinimumLiquidityKeeper)
	require.True(t, ok)
	require.NoError(t, keepertest.SetMinimumLiquidity(ctx, denomKeeper, constants.BaseCurrency, "10000000000"))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 math.LegacyZeroDec(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	_, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade77(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000000_000000))

	price1, err := k.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)

	tradeContext := types.TradeContext{
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx

		_, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	price2, err := k.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(price2))
}

func TestTrade78(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000_000000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	res1, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteSell(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))

	tradeContext.Context = ctx
	res2, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)
	require.True(t, res2.AmountReceived.LT(res1.AmountReceived))
}

func TestTrade79(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	_, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}))

	require.NoError(t, tradeContext.TradeBalances.Settle(ctx, k.BankKeeper))
}

func TestTrade80(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))

	minimumTradeAmount := math.NewInt(2_000000)
	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
		MinimumTradeAmount:  &minimumTradeAmount,
	}

	_, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}), "not enough liquidity")
}

func TestTrade81(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	_, err := k.SimulateBuy(tradeContext)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteBuy(tradeContext)
		return err
	}), "trade amount too small")
}

func TestTrade82(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))

	tradeContext := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	_, err := k.SimulateSell(tradeContext)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeContext.Context = innerCtx
		_, err = k.ExecuteSell(tradeContext)
		return err
	}), "trade amount too small")
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liqSum := k.GetLiquiditySum(ctx, denom).Int64()
		summedLiq := k.SumLiquidity(ctx, denom).Int64()
		funds := coins.AmountOf(denom).Int64()

		if liqSum != funds || summedLiq != funds {
			fmt.Println(denom)
			fmt.Println(fmt.Sprintf("liqSum: %v", liqSum))
			fmt.Println(fmt.Sprintf("summedLiq: %v", summedLiq))
			fmt.Println(fmt.Sprintf("funds: %v", funds))

			return false
		}
	}

	return true
}

func tradePoolEmpty(ctx context.Context, k dexkeeper.Keeper) error {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolTrade)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, coin := range coins {
		if coin.Amount.GT(math.ZeroInt()) {
			return fmt.Errorf("trade pool: %v %v", coin.Amount.String(), coin.Denom)
		}
	}

	return nil
}

func checkCache(ctx context.Context, k dexkeeper.Keeper) error {
	return k.CheckCache(ctx)
}
