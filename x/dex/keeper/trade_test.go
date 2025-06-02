package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/trading"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

// var priceTolerance = math.LegacyNewDecWithPrec(1001, 3) // 1.001
var priceTolerance = math.LegacyNewDec(1) // 1.001

func TestTradeSteps1(t *testing.T) {
	poolFrom := math.LegacyNewDec(1_000_000)
	poolBase := math.LegacyNewDec(4_000_000)
	poolTo := math.LegacyNewDec(1_000_000)
	maxPrice := math.LegacyNewDecWithPrec(101, 2)

	maxAmount := trading.CalculateMaximumBuyableByPrice(poolFrom, poolTo, maxPrice)
	amountToGive1, err := trading.ConstantProductTradeBuy(poolFrom, poolTo, maxAmount)
	require.NoError(t, err)

	intermediate, _ := trading.ConstantProductTradeBuy(poolBase, poolTo, maxAmount)
	poolBase = poolBase.Add(intermediate)
	amountToGive2, _ := trading.ConstantProductTradeBuy(poolFrom, poolBase, intermediate)

	pricePaid := amountToGive2.Quo(maxAmount) // C

	require.True(t, maxPrice.GTE(pricePaid))
	require.Equal(t, amountToGive1.TruncateInt().Int64(), amountToGive2.TruncateInt().Int64())
}

func TestSingleTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	setMovingLiqFixed(ctx, k)

	require.Equal(t, int64(10_000), k.GetPoolLiquidity(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(10_000), k.GetPoolLiquidity(ctx, constants.KUSD).Int64())

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(10_000), coins.AmountOf(constants.BaseCurrency).Int64())

	fee := math.LegacyZeroDec()

	ratio1, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), ratio1.Ratio)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(2_500),
		Fee:                 &fee,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           trading.FlatSell,
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.Equal(t, int64(2_500), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(10_000), tradeResult.AmountReceived().Int64())

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	liquidityPoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	liquidityPool := k.BankKeeper.SpendableCoins(ctx, liquidityPoolAcc.GetAddress())
	require.Equal(t, int64(0), liquidityPool.AmountOf(constants.BaseCurrency).Int64())

	require.Equal(t, int64(0), k.GetPoolLiquidity(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(12_500), k.GetPoolLiquidity(ctx, constants.KUSD).Int64())

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(0), coins.AmountOf(constants.BaseCurrency).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade1a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(10_000),
		Fee:                 &fee,
		Context:             ctx,
		TradeType:           types.TradeTypeBuy,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           trading.FlatBuy,
	}

	var (
		tradeResult trading.TradeResult
		err         error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	// amount is 9999 because 1 unit needs to stay in pool
	require.Equal(t, int64(2_500), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(10_000), tradeResult.AmountReceived().Int64())

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(2_500), tradeResult.AmountGiven().Int64())

	require.Equal(t, int64(0), k.GetPoolLiquidity(ctx, constants.BaseCurrency).Int64())
	require.Equal(t, int64(12_500), k.GetPoolLiquidity(ctx, constants.KUSD).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(10_000), coins.AmountOf(constants.BaseCurrency).Int64())

	offer := math.NewInt(10_000)
	fee := math.LegacyZeroDec()

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		FlatPrice:           trading.FlatSell,
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(10_000), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(20_000), coins.AmountOf(constants.BaseCurrency).Int64())
	require.Equal(t, int64(7_500), coins.AmountOf(constants.KUSD).Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	setMovingLiqFixed(ctx, k)

	offer := math.NewInt(2_500)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(10_000), liq.Int64())

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		FlatPrice:           trading.FlatSell,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(2_500), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(9600), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(400), tradeResult.FeeReceiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade3a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, int64(1_000_000), liq.Int64())

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		Context:             ctx,
		TradeType:           types.TradeTypeBuy,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		FlatPrice:           trading.FlatBuy,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(2_605), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(105), tradeResult.FeeGiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 5_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeCtx.TradeAmount = math.NewInt(1_250)
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(1_250), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(4_800), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(200), tradeResult.FeeReceiving().Int64())

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
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, tradeResult.AmountGiven())
	require.Equal(t, int64(2_400), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(0), tradeResult.FeeGiving().Int64())
	require.Equal(t, int64(100), tradeResult.FeeReceiving().Int64())

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
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, tradeResult.AmountGiven())
	require.Equal(t, int64(2_400), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(0), tradeResult.FeeGiving().Int64())
	require.Equal(t, int64(100), tradeResult.FeeReceiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 5_000))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(4, 2) // 4%

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, offer, tradeResult.AmountGiven())
	require.Equal(t, int64(2_400), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(0), tradeResult.FeeGiving().Int64())
	require.Equal(t, int64(100), tradeResult.FeeReceiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(1, 2)

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		TradeType:           types.TradeTypeSell,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(10_000), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(2_474), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(24), tradeResult.FeeReceiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade10(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	_, err := k.GetLiquidityPair(ctx, constants.KUSD)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(2, 3)

	tradeCtx := types.TradeContext{
		TradeAmount:         offer,
		Fee:                 &fee,
		MinimumTradeAmount:  &offer,
		Context:             ctx,
		CoinSource:          keepertest.Carol,
		CoinTarget:          keepertest.Carol,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(10_000), tradeResult.AmountGiven().Int64())
	require.Equal(t, int64(39_920), tradeResult.AmountReceived().Int64())
	require.Equal(t, int64(80), tradeResult.FeeReceiving().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
		Fee:                 &fee,
	}

	_, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res.PricePaidRounded()
	require.NoError(t, err)

	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(2_500),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res.PricePaidRounded()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000),
		MinimumTradeAmount:  zeroIntPtr(),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid1, _ := res1.PricePaid()
	tradeCtx.MaxPrice = &trading.MaxPriceData{
		MaxPrice:    pricePaid1,
		FeeIncluded: true,
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid2, err := res.PricePaidRounded()
	require.NoError(t, err)
	require.True(t, pricePaid1.GTE(pricePaid2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	maxPrice, err := math.LegacyNewDecFromStr("4.0856")
	require.NoError(t, err)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(5_000),
		MinimumTradeAmount:  zeroIntPtr(),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res.PricePaidRounded()
	require.NoError(t, err)

	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	tradeCtx.MaxPrice = &trading.MaxPriceData{
		MaxPrice:    maxPrice,
		FeeIncluded: true,
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid1))

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MinimumTradeAmount = zeroIntPtr()
		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid2, err := res2.PricePaidRounded()
	require.NoError(t, err)
	require.True(t, maxPrice.GTE(pricePaid2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000),
		MinimumTradeAmount:  zeroIntPtr(),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	price1, err := res1.PricePaid()
	require.NoError(t, err)

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    price1,
			FeeIncluded: true,
		}

		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	price2, err := res.PricePaidRounded()
	require.True(t, price2.LT(price1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(9_000_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MinimumTradeAmount = zeroIntPtr()
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res2.PricePaidRounded()
	require.NoError(t, err)

	pricePaid = pricePaid.Quo(priceTolerance)
	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade20x(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 5_151_753_574_086)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 8_968_155_459_761)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 98_236_914_564)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(10_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid1,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	pricePaid2, err := res2.PricePaidRounded()
	require.NoError(t, err)

	require.True(t, pricePaid2.LTE(pricePaid1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade20a(t *testing.T) {
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

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(9_000_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MinimumTradeAmount = zeroIntPtr()
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res2.PricePaidRounded()
	require.NoError(t, err)

	pricePaid = pricePaid.Quo(priceTolerance)
	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade20b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 5_151_753_574_086)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 8_968_155_459_761)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 98_236_914_564)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000_000))
	setMovingLiqFixed(ctx, k)

	r1, _ := math.LegacyNewDecFromStr("0.25")
	r2, _ := math.LegacyNewDecFromStr("0.25")

	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r1)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r2)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(9_000_000),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: "uwusdc",
		TradeBalances:       dexkeeper.NewTradeBalances(),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MinimumTradeAmount = zeroIntPtr()
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid, err := res2.PricePaidRounded()
	require.NoError(t, err)

	pricePaid = pricePaid.Quo(priceTolerance)
	require.True(t, maxPrice.GTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	tradeCtx := types.TradeContext{
		TradeAmount:         tradeAmount,
		MinimumTradeAmount:  &tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteBuy(tradeCtx)

		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	tradeCtx := types.TradeContext{
		TradeAmount:         tradeAmount,
		MinimumTradeAmount:  &tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		res, err = k.ExecuteBuy(tradeCtx)

		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	// numbers are not exactly equal due to rounding
	require.Equal(t, tradeAmount.Int64(), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	tradeCtx := types.TradeContext{
		TradeAmount:         tradeAmount,
		MinimumTradeAmount:  &tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		var err error
		tradeCtx.Context = innerCtx
		res, err = k.ExecuteBuy(tradeCtx)

		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	// numbers are not exactly equal due to rounding
	require.Equal(t, tradeAmount.Int64(), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	tradeCtx := types.TradeContext{
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
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
	tradeCtx := types.TradeContext{
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade29(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(10_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade30(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(600_000),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(innerCtx)
		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.Equal(t, int64(100_000), res1.AmountGiven.Int64())
	require.Equal(t, int64(396_039), res1.AmountReceived.Int64())

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(100_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    pricePaid,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestSingleTrade34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 400_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 100_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 100_000)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 400_000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 5_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeAmount := math.NewInt(10_000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	price1, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.OrdersCaches = k.NewOrdersCaches(ctx)
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice:    price1,
			FeeIncluded: true,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	price2, err := res2.PricePaidRounded()
	require.NoError(t, err)
	require.True(t, price2.LTE(price1))

	require.Equal(t, res1.AmountGiven.Int64(), res2.AmountGiven().Int64())
	require.Equal(t, res1.AmountReceived.Int64(), res2.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, math.NewInt(2_000_000), liq)
	liq = k.GetPoolLiquidity(ctx, constants.KUSD)
	require.Equal(t, math.NewInt(2_000_000), liq)

	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000000",
	})

	require.NoError(t, err)
	amountReceived, _ := strconv.Atoi(res.AmountReceived)
	require.Equal(t, 243_658, amountReceived)

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

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
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

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
	require.Equal(t, liq, math.NewInt(6_000_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	liq := k.GetPoolLiquidity(ctx, constants.BaseCurrency)
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
	require.Equal(t, "243658", res.AmountReceived)

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
	require.Equal(t, int64(122), coins[0].Amount.Int64())

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

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 50_000))

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "2_000_000",
	})
	require.NoError(t, err)

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
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

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "2000",
	})
	require.NoError(t, err)

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
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
	require.Equal(t, 1, len(k.LiquidityIterator(ctx, constants.KUSD).GetAll()))

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

	require.Equal(t, 2, len(k.LiquidityIterator(ctx, constants.KUSD).GetAll()))
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade13(t *testing.T) {
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

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000))

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    "0.01",
			FeeIncluded: false,
		},
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

	response, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    "0.01",
			FeeIncluded: false,
		},
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

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100_000",
	})

	require.NoError(t, err)
	require.NotNil(t, response)

	amountReceived, _ := strconv.Atoi(response.AmountReceived)
	amountGiven, _ := strconv.Atoi(response.AmountGiven)

	price := float64(amountGiven) / float64(amountReceived)
	require.Equal(t, 4.014129736673089, price)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000))

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "4000",
	})

	require.NoError(t, err)
	require.Equal(t, "4000", response.AmountGiven)
	require.Equal(t, "998", response.AmountReceived)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5_000))

	response, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
	})

	require.NoError(t, err)
	require.Equal(t, "10000", response.AmountGiven)
	require.Equal(t, "2496", response.AmountReceived)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	setMovingLiqFixed(ctx, k)

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)

	ratio1, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1_000_000",
	})
	require.NoError(t, err)

	ratio2, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)
	require.True(t, ratio1.Ratio.GT(ratio2.Ratio))

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)
	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func getCoin(coins []sdk.Coin, denom string) sdk.Coin {
	for _, coin := range coins {
		if coin.Denom == denom {
			return coin
		}
	}

	return sdk.Coin{}
}

func TestTrade24(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Carol,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
	})
	require.NoError(t, err)
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
		MaxPrice: &types.MaxPrice{
			MaxPrice:    maxPrice.String(),
			FeeIncluded: false,
		},
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

func TestTrade32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	fee := math.LegacyNewDecWithPrec(4, 2)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
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

	fee := math.LegacyNewDecWithPrec(4, 2)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.BaseCurrency,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
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

func TestTrade35(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))

	fee := math.LegacyNewDecWithPrec(4, 2)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
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

	liq1_uwusdc := k.GetPoolLiquidity(ctx, "uwusdc")
	liq1_ukusd := k.GetPoolLiquidity(ctx, constants.KUSD)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(tradeAmount),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
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

	liq2_uwusdc := k.GetPoolLiquidity(ctx, "uwusdc")
	liq2_ukusd := k.GetPoolLiquidity(ctx, constants.KUSD)

	require.Equal(t, funds1_uwusdc-tradeAmount, funds2_uwusdc)
	require.Equal(t, funds1_ukusd+tradeAmount, funds2_ukusd)

	require.Equal(t, liq1_uwusdc.Int64()+tradeAmount, liq2_uwusdc.Int64())
	require.Equal(t, liq1_ukusd.Int64()-tradeAmount, liq2_ukusd.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
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
	fee := math.LegacyNewDecWithPrec(1, 2)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(tradeAmount),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		FlatPrice:           trading.FlatSell,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
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
	require.Equal(t, 1, len(liqOther))
	require.Equal(t, int64(9_751), liqOther[0].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade39(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	res1, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1_000000",
	})
	require.NoError(t, err)

	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1_000000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    fmt.Sprintf("%.8f", price1),
			FeeIncluded: false,
		},
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade39a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	setMovingLiqFixed(ctx, k)

	res1, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "100_000000_000000",
	})
	require.NoError(t, err)

	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "100_000000_000000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    fmt.Sprintf("%.8f", price1),
			FeeIncluded: false,
		},
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade39b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))
	setMovingLiqFixed(ctx, k)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 &fee,
	}

	res, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)
	simulatedPrice, err := res.PricePaid()
	require.NoError(t, err)

	res1, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_000000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    fmt.Sprintf("%.8f", simulatedPrice),
			FeeIncluded: true,
		},
	})
	require.NoError(t, err)

	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_0000000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    fmt.Sprintf("%.8f", price1),
			FeeIncluded: false,
		},
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade40(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	res1, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "500_000",
	})
	require.NoError(t, err)

	amountGiven1, _ := strconv.ParseFloat(res1.AmountGiven, 64)
	amountReceived1, _ := strconv.ParseFloat(res1.AmountReceived, 64)
	price1 := amountGiven1 / amountReceived1

	_, err = keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "500_000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    fmt.Sprintf("%.8f", price1),
			FeeIncluded: false,
		},
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade41(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 200_000)
	res, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "100_000",
	})
	require.NoError(t, err)

	amountGiven, _ := strconv.Atoi(res.AmountGiven)
	require.LessOrEqual(t, amountGiven, 111_223)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade42(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	pool := []keepertest.AddLiquidityData{
		{
			"ukopi", math.NewInt(5223913832979),
		},
		{
			"ukusd", math.NewInt(479921845),
		},
		{
			"ucwusdc", math.NewInt(38585199076),
		},
	}

	ratios := map[string]string{
		"ukusd":   "0.196939223849083103",
		"ucwusdc": "0.211427997614305922",
	}

	balance := map[string]int64{
		"ukopi":   5223913832979,
		"ukusd":   3899152227,
		"ucwusdc": 38585199076,
	}

	for denom, amount := range balance {
		keepertest.AddFunds(ctx, t, k.BankKeeper, denom, keepertest.Dave, amount)
	}

	keepertest.SetLiquidity(ctx, k.BankKeeper, k, t, pool)
	require.True(t, liquidityBalanced(ctx, k))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		for denom, ratio := range ratios {
			r, _ := math.LegacyNewDecFromStr(ratio)
			k.DenomKeeper.SetRatio(innerCtx, denomtypes.Ratio{Denom: denom, Ratio: r})
		}

		return nil
	}))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))
	require.True(t, liquidityBalanced(ctx, k))

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
		Amount:         "10_000_000",
	})
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade43(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	pool := []keepertest.AddLiquidityData{
		{
			"ukopi", math.NewInt(49994130281493),
		},
		{
			"inj", math.NewInt(829132866175164798),
		},
		{
			"ukusd", math.NewInt(1740434604),
		},
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
		MinimumTradeAmount: "1000000000000000",
		MaxPrice: &types.MaxPrice{
			MaxPrice:    "48057996063.181141677468913534",
			FeeIncluded: false,
		},
	})
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade44(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Alice,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "10_000",
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade44a(t *testing.T) {
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
	require.ErrorContains(t, err, "not enough usage liquidity for address")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade44b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Bob,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "uwusdc",
		Amount:             "10_000_000",
		MinimumTradeAmount: "0",
	})
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade45(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	acc1, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	acc2, _ := sdk.AccAddressFromBech32(keepertest.Dave)
	accLiq := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)

	amount := int64(1_000_000)
	coins := sdk.NewCoins(
		sdk.NewCoin(constants.BaseCurrency, math.NewInt(amount)),
		sdk.NewCoin(constants.KUSD, math.NewInt(amount*2)),
		sdk.NewCoin("uwusdc", math.NewInt(amount)),
	)

	_ = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc1, types.PoolReserve, coins)
	_ = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolReserve, acc2, coins)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, constants.BaseCurrency, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, constants.KUSD, amount))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Dave, "uwusdc", amount))
	require.True(t, liquidityBalanced(ctx, k))

	ratioKUSD, _ := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	ratioUSDC, _ := k.DenomKeeper.GetRatio(ctx, "uwusdc")

	require.True(t, ratioKUSD.Ratio.Equal(ratioUSDC.Ratio))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, amount)
	res, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Alice,
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
	require.Equal(t, 1_000_000, amountGiven)
	require.Equal(t, 908_181, amountReceived)

	require.Equal(t, int64(2_000_000), k.GetLiquidityByAddress(ctx, constants.KUSD, keepertest.Dave).Int64())
	require.Equal(t, int64(90_910), k.GetLiquidityByAddress(ctx, "uwusdc", keepertest.Dave).Int64())

	require.Equal(t, int64(2_000_000), k.BankKeeper.SpendableCoin(ctx, accLiq.GetAddress(), constants.KUSD).Amount.Int64())
	require.Equal(t, int64(90_910), k.BankKeeper.SpendableCoin(ctx, accLiq.GetAddress(), "uwusdc").Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
		MaxPrice: &types.MaxPrice{
			MaxPrice:    maxPrice,
			FeeIncluded: false,
		},
	})

	require.Error(t, err)
}

func TestTrade48a(t *testing.T) {
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

	require.Error(t, err)
}

func TestTrade48b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Bob, 20000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4467855392339))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 63286126324))
	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "10002952303552758024"))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.335987740910407578")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	r, _ = math.LegacyNewDecFromStr("16534992150.086859516468639030")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", r)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:            keepertest.Bob,
		DenomGiving:        constants.KUSD,
		DenomReceiving:     "inj",
		Amount:             "20000000",
		MinimumTradeAmount: "0",
	})

	require.NoError(t, err)

	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade49a(t *testing.T) {
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

	require.Error(t, err)
}

func TestTrade49b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 3140427631280+1326651942499)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 12622692067+5537967689)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4467855392339))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 63286126324))
	require.NoError(t, keepertest.AddLiquidityString(ctx, msg, keepertest.Alice, "inj", "10002952303552758024"))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.335987740910407578")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	r, _ = math.LegacyNewDecFromStr("16534992150.086859516468639030")
	keepertest.SetRatio(ctx, k.DenomKeeper, "inj", r)

	_, err := keepertest.Buy(ctx, msg, &types.MsgBuy{
		Creator:            keepertest.Bob,
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

	maximumAvailable := int64(10_000)
	tradeCtx := types.TradeContext{
		TradeAmount:            math.NewInt(100_000),
		MaximumAvailableAmount: math.NewInt(maximumAvailable),
		CoinSource:             keepertest.Bob,
		CoinTarget:             keepertest.Bob,
		TradeDenomGiving:       constants.BaseCurrency,
		TradeDenomReceiving:    constants.KUSD,
		OrdersCaches:           k.NewOrdersCaches(ctx),
		TradeBalances:          dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.GreaterOrEqual(t, maximumAvailable, res.AmountGiven().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade51(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.Equal(t, int64(404_445), res.AmountGiven().Int64())
	require.Equal(t, int64(100_000), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade52(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(100_000), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade53(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(100_000), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade54(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(9_989), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade55(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(9_987), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade56(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(9_989), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade56a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(9_989), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade57(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
	require.Equal(t, int64(100_000), res.AmountReceived().Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	price, err := res.PricePaidRounded()
	require.NoError(t, err)
	return price
}

func TestTrade59(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(10_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade60(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)

	maxPrice, _ := math.LegacyNewDecFromStr("1.11")

	tradeCtx := types.TradeContext{
		MaxPrice: &trading.MaxPriceData{
			MaxPrice: maxPrice,
		},
		TradeAmount:         math.NewInt(100_000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "inj",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
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

func TestTrade62(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 40_000_000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Bob, 1000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 40_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000_000_000))
	setMovingLiqFixed(ctx, k)

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	tradeBalances := dexkeeper.NewTradeBalances()

	var res1 trading.TradeResult
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
			TradeBalances:       tradeBalances,
		})

		return err
	}))

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       tradeBalances,
	}

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		var err error
		res2, err = k.ExecuteSell(tradeCtx)

		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	price1, _ := res1.PricePaidRounded()
	price2, _ := res2.PricePaidRounded()

	require.True(t, res2.AmountReceived().LT(res1.AmountReceived()))
	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
		var res trading.TradeResult
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

		require.True(t, liquidityBalanced(ctx, k))
		require.NoError(t, tradePoolEmpty(ctx, k))
		require.NoError(t, checkCache(ctx, k))
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

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade66(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade67(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade68(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade69(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade70(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	fee := math.LegacyNewDecWithPrec(1, 3)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &fee,
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	maxPrice, _ := res1.PricePaid()

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: maxPrice,
		}

		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade71(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000))
	setMovingLiqFixed(ctx, k)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	maxPrice, _ := math.LegacyNewDecFromStr("0.1")

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := k.ExecuteSell(types.TradeContext{
			MaxPrice: &trading.MaxPriceData{
				MaxPrice: maxPrice,
			},
			Context:             innerCtx,
			TradeAmount:         math.NewInt(100_000_000000),
			CoinSource:          keepertest.Bob,
			CoinTarget:          keepertest.Bob,
			TradeDenomGiving:    constants.KUSD,
			TradeDenomReceiving: constants.BaseCurrency,
			OrdersCaches:        k.NewOrdersCaches(ctx),
			TradeBalances:       dexkeeper.NewTradeBalances(),
			Fee:                 zeroDecPtr(),
		})
		return err
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade72(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))
	setMovingLiqFixed(ctx, k)

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
			Fee:                 zeroDecPtr(),
		})
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade73(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))
	setMovingLiqFixed(ctx, k)

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
			Fee:                 zeroDecPtr(),
		})
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade74(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4114494_580260)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 52875_551701)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4114494_580260))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 52875_551701))
	setMovingLiqFixed(ctx, k)

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
			Fee:                 zeroDecPtr(),
		})
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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
			Fee:                 zeroDecPtr(),
		})
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade76(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50037598073771)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 4925)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 68529039897008)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1000_005000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1000_005000))

	r, _ := math.LegacyNewDecFromStr("0.25")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	denomKeeper, ok := k.DenomKeeper.(keepertest.SetMinimumLiquidityKeeper)
	require.True(t, ok)
	require.NoError(t, keepertest.SetMinimumLiquidity(ctx, denomKeeper, constants.BaseCurrency, "10000000000"))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	_, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
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

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)

	tradeCtx := types.TradeContext{
		TradeAmount:         math.NewInt(1_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)

	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade78(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000_000000))
	setMovingLiqFixed(ctx, k)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	tradeCtx.Context = ctx
	res2, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)
	require.True(t, res2.AmountReceived.LT(res1.AmountReceived))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade79(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(999_999),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	_, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade80(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))
	setMovingLiqFixed(ctx, k)

	minimumTradeAmount := math.NewInt(2_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
		MinimumTradeAmount:  &minimumTradeAmount,
	}

	_, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}), "not enough buyable liquidity")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade81(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	setMovingLiqFixed(ctx, k)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	_, err := k.SimulateBuyFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteBuy(tradeCtx)
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade82(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000000_000000))
	setMovingLiqFixed(ctx, k)

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(2_000000),
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	_, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.ExecuteSell(tradeCtx)
		return err
	}), "trade amount too small")

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

type AddDenomInterface interface {
	CreateDexDenom(ctx context.Context, name, factorStr, minLiquidityStr, minOrderSizeStr, minVirtualLiquidityStr string, exponent uint64) (denomtypes.DexDenom, denomtypes.Ratio, error)
	DexAddDenom(ctx context.Context, dexDenom denomtypes.DexDenom, ratio denomtypes.Ratio) error
}

func TestTrade83(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	denomKeeper := k.DenomKeeper.(AddDenomInterface)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable1", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable2", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable2", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable1", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable2", 1_000000))

	ratio11, _ := k.DenomKeeper.GetRatio(ctx, "stable1")
	ratio12, _ := k.DenomKeeper.GetRatio(ctx, "stable2")

	priceStable1, err := k.DenomKeeper.CalculatePrice(ctx, "stable1", "stable2")
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    "stable1",
		DenomReceiving: "stable2",
		Amount:         "1_000000",
	})
	require.NoError(t, err)

	ratio21, _ := k.DenomKeeper.GetRatio(ctx, "stable1")
	ratio22, _ := k.DenomKeeper.GetRatio(ctx, "stable2")

	priceStable2, err := k.DenomKeeper.CalculatePrice(ctx, "stable1", "stable2")
	require.NoError(t, err)

	require.True(t, priceStable2.GT(priceStable1))
	require.True(t, ratio11.Ratio.LT(ratio21.Ratio))
	require.True(t, ratio12.Ratio.GT(ratio22.Ratio))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade84(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	denomKeeper := k.DenomKeeper.(AddDenomInterface)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable1", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "funky1", "1ukusd", "1_000_000", "1_000_000", "1_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "funky1", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Bob, 1_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable1", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "funky1", 1_000000))

	ratio11, _ := k.DenomKeeper.GetRatio(ctx, "stable1")
	ratio12, _ := k.DenomKeeper.GetRatio(ctx, "funky1")

	priceStable1, err := k.DenomKeeper.CalculatePrice(ctx, "stable1", "funky1")
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    "stable1",
		DenomReceiving: "funky1",
		Amount:         "1_000000",
	})
	require.NoError(t, err)

	ratio21, _ := k.DenomKeeper.GetRatio(ctx, "stable1")
	ratio22, _ := k.DenomKeeper.GetRatio(ctx, "funky1")

	priceStable2, err := k.DenomKeeper.CalculatePrice(ctx, "stable1", "funky1")
	require.NoError(t, err)

	require.True(t, priceStable2.GT(priceStable1))
	require.True(t, ratio11.Ratio.LT(ratio21.Ratio))
	require.True(t, ratio12.Ratio.GT(ratio22.Ratio))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade85(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)
	denomKeeper := k.DenomKeeper.(AddDenomInterface)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable1", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable2", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "stable3", "1ukusd", "1_000_000", "1_000_000", "1_000_000_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denom, ratio, err := denomKeeper.CreateDexDenom(innerCtx, "funky1", "1ukusd", "1_000_000", "1_000_000", "1_000_000", 6)
		if err != nil {
			return nil
		}

		return denomKeeper.DexAddDenom(innerCtx, denom, ratio)
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable1", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable2", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "stable3", keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "funky1", keepertest.Alice, 1_000000_000000)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable1", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable2", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "stable3", 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "funky1", 1_000000))

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Alice,
		DenomGiving:    "stable1",
		DenomReceiving: "stable2",
		Amount:         "1_000000",
	})
	require.Error(t, err)
}

func TestTrade86(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 5326805883)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 77569727)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 70_000000))

	amount := math.NewInt(50_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: pricePaid,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid2, err := res2.PricePaidRounded()
	require.NoError(t, err)

	require.True(t, pricePaid2.LTE(pricePaid))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade87(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 5326805883)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 77569727)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 5000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 70_000000))

	r := math.LegacyNewDecWithPrec(5, 1)
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r)

	amount := math.NewInt(50_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)

	var res2 trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: pricePaid1,
		}

		res2, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid2, err := res2.PricePaidRounded()
	require.NoError(t, err)
	require.True(t, pricePaid2.LTE(pricePaid1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade88(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 5326805883)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 77569727)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000000))

	amount := math.NewInt(5_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MinimumTradeAmount = &res1.AmountGiven
		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade89(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 5326805883)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 77569727)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 70_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 70_000000))

	amount := math.NewInt(50_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	res1, err := k.SimulateSellFromTradeContext(tradeCtx)
	require.NoError(t, err)

	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.MaxPrice = &trading.MaxPriceData{
			MaxPrice: pricePaid1,
		}

		_, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	pricePaid2, err := res1.PricePaid()
	require.NoError(t, err)

	require.True(t, pricePaid2.LTE(pricePaid1))

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, tradePoolEmpty(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestTrade90(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 9_973955_830468)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 86_019_995791)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 8_973955_830468))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 7019_995791))

	r11, _ := math.LegacyNewDecFromStr("0.242286405309279926")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r11)
	r21, _ := math.LegacyNewDecFromStr("0.238085503436972988")
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r21)

	amount := math.NewInt(50_000000)
	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	_, err := k.DenomKeeper.GetRatio(ctx, constants.KUSD)
	require.NoError(t, err)
	_, err = k.DenomKeeper.GetRatio(ctx, "uwusdc")
	require.NoError(t, err)
}

func TestTrade91(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 9_973955_830468)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 86_019_995791)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 8_973955_830468))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 7019_995791))

	r11, _ := math.LegacyNewDecFromStr("0.242286405309279926")
	keepertest.SetRatio(ctx, k.DenomKeeper, constants.KUSD, r11)
	r21, _ := math.LegacyNewDecFromStr("0.238085503436972988")
	keepertest.SetRatio(ctx, k.DenomKeeper, "uwusdc", r21)

	amount := math.NewInt(50_000000)
	maxPrice := math.LegacyNewDecWithPrec(1, 4)
	tradeCtx := types.TradeContext{
		MaxPrice: &trading.MaxPriceData{
			MaxPrice: maxPrice,
		},
		Context:             ctx,
		TradeAmount:         amount,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 zeroDecPtr(),
	}

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx

		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}))
}

func TestTrade92(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}), "not enough usage liquidity for address")
}

func TestTrade93(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 25_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteBuy(tradeCtx)
		return err
	}), "not enough usage liquidity for address")
}

func TestTrade94(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}), "not enough usage liquidity for address")
}

func TestTrade95(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 25_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 25_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}), "not enough usage liquidity for address")
}

func TestTrade96(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "uwusdc", 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 21_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(10_000000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))

	tradeCtx.TradeAmount = math.NewInt(1_000000)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}))
}

func TestTrade97(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.BaseCurrency, 100_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, constants.KUSD, 3_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "uwusdc", 4_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 22_000000))

	tradeCtx := types.TradeContext{
		Context:             ctx,
		TradeAmount:         math.NewInt(100_000000),
		CoinSource:          keepertest.Alice,
		CoinTarget:          keepertest.Alice,
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		Fee:                 zeroDecPtr(),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err := k.ExecuteSell(tradeCtx)
		return err
	}), "not enough usage liquidity for address")
}

func TestTrade98(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000000))

	tradeCtx := types.TradeContext{
		Context:       ctx,
		TradeAmount:   math.NewInt(1_000000),
		CoinSource:    keepertest.Bob,
		CoinTarget:    keepertest.Bob,
		Fee:           zeroDecPtr(),
		TradeBalances: dexkeeper.NewTradeBalances(),
		OrdersCaches:  k.NewOrdersCaches(ctx),
	}

	var res trading.TradeResult
	for range 10 {
		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			tradeCtx.Context = innerCtx
			tradeCtx.TradeDenomGiving = "uwusdc"
			tradeCtx.TradeDenomReceiving = constants.KUSD

			var err error
			res, err = k.ExecuteSell(tradeCtx)
			return err
		}))

		amount := res.AmountReceived().ToLegacyDec().Quo(math.LegacyNewDec(1_000000))
		require.True(t, amount.LT(math.LegacyOneDec()))

		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			tradeCtx.Context = innerCtx
			tradeCtx.TradeDenomGiving = constants.KUSD
			tradeCtx.TradeDenomReceiving = "uwusdc"
			tradeCtx.TradeAmount = res.AmountReceived()

			var err error
			res, err = k.ExecuteSell(tradeCtx)
			return err
		}))

		amount = res.AmountReceived().ToLegacyDec().Quo(math.LegacyNewDec(1_000000))
		require.True(t, amount.LT(math.LegacyOneDec()))
	}
}

func TestTrade99(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000000))

	tradeCtx := types.TradeContext{
		Context:       ctx,
		TradeAmount:   math.NewInt(1_000000),
		CoinSource:    keepertest.Bob,
		CoinTarget:    keepertest.Bob,
		Fee:           zeroDecPtr(),
		TradeBalances: dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	for range 10 {
		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			tradeCtx.Context = innerCtx
			tradeCtx.OrdersCaches = k.NewOrdersCaches(innerCtx)
			tradeCtx.TradeDenomGiving = "uwusdc"
			tradeCtx.TradeDenomReceiving = constants.KUSD

			var err error
			res, err = k.ExecuteSell(tradeCtx)
			return err
		}))

		require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
		amount := res.AmountReceived().ToLegacyDec().Quo(math.LegacyNewDec(1_000000))
		require.True(t, amount.LT(math.LegacyOneDec()))

		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			tradeCtx.Context = innerCtx
			tradeCtx.OrdersCaches = k.NewOrdersCaches(innerCtx)
			tradeCtx.TradeDenomGiving = constants.KUSD
			tradeCtx.TradeDenomReceiving = "uwusdc"
			tradeCtx.TradeAmount = res.AmountReceived()

			var err error
			res, err = k.ExecuteSell(tradeCtx)
			return err
		}))

		require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))
		amount = res.AmountReceived().ToLegacyDec().Quo(math.LegacyNewDec(1_000000))
		require.True(t, amount.LT(math.LegacyOneDec()))
	}
}

func TestTrade100(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000000))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		amount := math.LegacyNewDec(1_000_000000)
		k.SetMovingLiquidity(innerCtx, constants.BaseCurrency, amount)
		k.SetMovingLiquidity(innerCtx, constants.KUSD, amount)
		k.SetMovingLiquidity(innerCtx, "uwusdc", amount)
		return nil
	}))

	tradeAmount := math.NewInt(1_000000)
	tradeCtx := types.TradeContext{
		Context:       ctx,
		TradeAmount:   tradeAmount,
		CoinSource:    keepertest.Bob,
		CoinTarget:    keepertest.Bob,
		Fee:           zeroDecPtr(),
		TradeBalances: dexkeeper.NewTradeBalances(),
	}

	var res trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.TradeDenomGiving = "uwusdc"
		tradeCtx.TradeDenomReceiving = constants.KUSD

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 1_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000000))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeCtx.TradeDenomGiving = constants.KUSD
		tradeCtx.TradeDenomReceiving = "uwusdc"
		tradeCtx.TradeAmount = res.AmountReceived()

		var err error
		res, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, res.AmountReceived().LT(tradeAmount))
}

func TestUpdateRatioToBaseOneStep1a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig"
		usmall = "usmall"
	)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	liqBase := k.GetEffectiveLiquidity(ctx, constants.BaseCurrency)

	changeBase := math.LegacyOneDec().Neg()
	changeOther := math.LegacyOneDec()
	tradeValue := math.LegacyOneDec()

	changes := types.AmountsMap{}
	changes.Add(constants.BaseCurrency, changeBase)
	changes.Add(ubig, changeOther)

	changeRatio := tradeValue.Quo(liqBase)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		for _, ratio := range k.DenomKeeper.GetAllRatios(innerCtx) {
			if ratio.Denom != ubig && ratio.Denom != usmall {
				continue
			}

			changeOther = changes.AmountOf(ratio.Denom)
			if err := k.UpdateRatioToBase(innerCtx, ratio, liqBase, changeRatio, changeBase, changeOther); err != nil {
				return err
			}
		}

		return nil
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	require.True(t, rBig.Ratio.GT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep1b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig"
		usmall = "usmall"
	)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	liqBase := k.GetEffectiveLiquidity(ctx, constants.BaseCurrency)

	changeBase := math.LegacyOneDec()
	changeOther := math.LegacyOneDec().Neg()
	tradeValue := math.LegacyOneDec()

	changes := types.AmountsMap{}
	changes.Add(constants.BaseCurrency, changeBase)
	changes.Add(ubig, changeOther)

	changeRatio := tradeValue.Quo(liqBase)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		for _, ratio := range k.DenomKeeper.GetAllRatios(innerCtx) {
			if ratio.Denom != ubig && ratio.Denom != usmall {
				continue
			}

			changeOther = changes.AmountOf(ratio.Denom)
			if err := k.UpdateRatioToBase(innerCtx, ratio, liqBase, changeRatio, changeBase, changeOther); err != nil {
				return err
			}
		}

		return nil
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	require.True(t, rBig.Ratio.LT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep2a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeBase := math.OneInt()
	changeOther := math.OneInt().Neg()
	tradeValue := math.LegacyNewDec(10_000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, constants.BaseCurrency, ubig, tradeValue, changeBase, changeOther)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	require.True(t, rBig.Ratio.LT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep2b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeBase := math.OneInt().Neg()
	changeOther := math.OneInt()
	tradeValue := math.LegacyNewDec(10_000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, ubig, constants.BaseCurrency, tradeValue, changeOther, changeBase)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	require.True(t, rBig.Ratio.GT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep2c(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeBase := math.OneInt().Neg()
	changeOther := math.OneInt()
	tradeValue := math.LegacyNewDec(1000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, usmall, constants.BaseCurrency, tradeValue, changeOther, changeBase)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	fmt.Println(rBig.Ratio.String())
	fmt.Println(rSmall.Ratio.String())

	require.True(t, rBig.Ratio.LT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep2d(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeBase := math.OneInt()
	changeOther := math.OneInt().Neg()
	tradeValue := math.LegacyNewDec(1000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, constants.BaseCurrency, usmall, tradeValue, changeBase, changeOther)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	fmt.Println(rBig.Ratio.String())
	fmt.Println(rSmall.Ratio.String())

	require.True(t, rBig.Ratio.GT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep3a(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeBig := math.OneInt()
	changeSmall := math.OneInt().Neg()
	tradeValue := math.LegacyNewDec(1000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, ubig, usmall, tradeValue, changeBig, changeSmall)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)

	rSmall.Ratio = math.LegacyOneDec().Quo(rSmall.Ratio)
	require.True(t, rBig.Ratio.LT(rSmall.Ratio))
}

func TestUpdateRatioToBaseOneStep3b(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		ubig   = "ubig2"
		usmall = "usmall2"
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		denomK := k.DenomKeeper.(denomkeeper.Keeper)
		_ = denomK.DexUpdateMinimumTradeLiquidity(innerCtx, constants.BaseCurrency, "1000")
		return nil
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, ubig, keepertest.Alice, 1000)
	keepertest.AddFunds(ctx, t, k.BankKeeper, usmall, keepertest.Alice, 100)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, ubig, 1000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, usmall, 100))

	keepertest.SetRatio(ctx, k.DenomKeeper, ubig, math.LegacyOneDec())
	keepertest.SetRatio(ctx, k.DenomKeeper, usmall, math.LegacyOneDec())

	changeSmall := math.OneInt()
	changeBig := math.OneInt().Neg()
	tradeValue := math.LegacyNewDec(1000)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.UpdateRatiosToBase(innerCtx, usmall, ubig, tradeValue, changeSmall, changeBig)
	}))

	rBig, _ := k.DenomKeeper.GetRatio(ctx, ubig)
	rSmall, _ := k.DenomKeeper.GetRatio(ctx, usmall)
	rSmall.Ratio = math.LegacyOneDec().Quo(rSmall.Ratio)

	require.True(t, rBig.Ratio.GT(rSmall.Ratio))
}

func TestOSMO(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	const (
		uosmo = "uosmo"
		uusdc = "uusdc"
	)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 49996770588553)
	keepertest.AddFunds(ctx, t, k.BankKeeper, uosmo, keepertest.Alice, 91034258338)
	keepertest.AddFunds(ctx, t, k.BankKeeper, uusdc, keepertest.Alice, 290)
	keepertest.AddFunds(ctx, t, k.BankKeeper, uosmo, keepertest.Bob, 91034258338)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 49996770588553))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, uosmo, 1034258338))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, uusdc, 290))

	rOSMO, _ := math.LegacyNewDecFromStr("0.060617941395790768")
	rUSDC, _ := math.LegacyNewDecFromStr("0.199054308825968250")
	keepertest.SetRatio(ctx, k.DenomKeeper, uosmo, rOSMO)
	keepertest.SetRatio(ctx, k.DenomKeeper, uusdc, rUSDC)

	_, err := keepertest.Sell(ctx, msg, &types.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    uosmo,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "2500_000000",
	})
	require.NoError(t, err)
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	if ok := liquidityPoolBalanced(ctx, k); !ok {
		return false
	}

	if ok := liquidityPositionsBalanced(ctx, k); !ok {
		return false
	}

	return true
}

func liquidityPositionsBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	for _, address := range k.GetLiquidityPositionsAddresses(ctx) {
		if ok := liquidityPositionsBalancedAddress(ctx, k, address); !ok {
			return false
		}
	}

	return true
}

func liquidityPositionsBalancedAddress(ctx context.Context, k dexkeeper.Keeper, address string) bool {
	denomPositionSum := k.GetLiquidityAddressSums(ctx, address)
	positions, _ := k.LiquidityPositionForAddress(ctx, &types.QueryLiquidityPositionForAddressRequest{Address: address})
	denomPositionLiq := sdk.NewCoins()

	for _, position := range positions.LiquidityPositions {
		for _, entry := range position.LiquidityPositionEntries {
			amount, _ := math.NewIntFromString(entry.UserAmount)
			denomPositionLiq = denomPositionLiq.Add(sdk.NewCoin(entry.Denom, amount))
		}
	}

	return denomPositionSum.Equal(denomPositionLiq)
}

func liquidityPoolBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		summedLiq := k.SumLiquidity(ctx, denom).Int64()
		funds := coins.AmountOf(denom).Int64()

		if summedLiq != funds {
			fmt.Println(denom)
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

func setMovingLiqFixed(ctx context.Context, k dexkeeper.Keeper) {
	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
		balance := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

		for _, denom := range k.DenomKeeper.Denoms(ctx) {
			amount := balance.AmountOf(denom)
			k.SetMovingLiquidity(innerCtx, denom, amount.ToLegacyDec())
		}

		return nil
	})
}

func zeroIntPtr() *math.Int {
	zeroInt := math.ZeroInt()
	return &zeroInt
}

func zeroDecPtr() *math.LegacyDec {
	zeroDec := math.LegacyZeroDec()
	return &zeroDec
}
