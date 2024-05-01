package keeper_test

import (
	"context"
	"fmt"
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

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	require.Nil(t, maximum)
}

func TestCalculateSingleMaximumTradableAmount2(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(500)
	virtualTo := math.LegacyNewDec(500)

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	require.NotNil(t, maximum)

	receive := dexkeeper.ConstantProductTrade(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum)
	require.Equal(t, receive, actualTo)
}

func TestCalculateSingleMaximumTradableAmount3(t *testing.T) {
	actualFrom := math.LegacyNewDec(1000)
	virtualFrom := math.LegacyNewDec(0)

	actualTo := math.LegacyNewDec(100)
	virtualTo := math.LegacyNewDec(900)

	maximum := dexkeeper.CalculateSingleMaximumTradableAmount(actualFrom, actualTo, virtualFrom, virtualTo)
	require.NotNil(t, maximum)

	receive := dexkeeper.ConstantProductTrade(actualFrom.Add(virtualFrom), actualTo.Add(virtualTo), *maximum)
	require.Equal(t, receive, actualTo)
}

func TestSingleTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100)
	require.Nil(t, err)

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(utils.BaseCurrency).Int64())

	offer := math.NewInt(100)
	fee := math.LegacyZeroDec()

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		CoinSource:       addr,
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	_, _, _, err = k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amount))
	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(0), coins.AmountOf(utils.BaseCurrency).Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100)
	require.Nil(t, err)

	dexAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(100), coins.AmountOf(utils.BaseCurrency).Int64())

	offer := math.NewInt(100)
	fee := math.LegacyZeroDec()

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      offer,
		CoinSource:       addr,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	usedAmount, receivedAmount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amount))
	require.NoError(t, err)

	coins = k.BankKeeper.SpendableCoins(ctx, dexAcc)
	require.Equal(t, int64(200), coins.AmountOf(utils.BaseCurrency).Int64())
	require.Equal(t, int64(0), coins.AmountOf("ukusd").Int64())
	require.Equal(t, int64(100), usedAmount.Int64())
	require.Equal(t, int64(100), receivedAmount.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100)
	require.Nil(t, err)

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, int64(100), liq.Int64())

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	amountUsed, amountReceived, feePaid, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	_, amountReceived, _, err = k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amountReceived))
	require.NoError(t, err)

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50)
	require.Nil(t, err)

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      offer,
		MaxPrice:         nil,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		AllowIncomplete:  false,
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	amountUsed, amountReceived, feePaid, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	_, amountReceived, _, err = k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amountReceived))
	require.NoError(t, err)

	require.Equal(t, offer, amountUsed)
	require.Equal(t, math.NewInt(98), amountReceived)
	require.Equal(t, math.NewInt(2), feePaid)

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.Nil(t, err)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Carol)

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	options := types.TradeOptions{
		GivenAmount:      offer,
		CoinSource:       acc,
		CoinTarget:       acc,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	amountUsed, amountReceived, feePaid, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amount))
	require.NoError(t, err)

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(200), feePaid.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.Nil(t, err)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Carol)

	offer := math.NewInt(10_000)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	options := types.TradeOptions{
		GivenAmount:      offer,
		CoinSource:       acc,
		CoinTarget:       acc,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	amountUsed, amountReceived, feePaid, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amount))
	require.NoError(t, err)

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(9_800), amountReceived.Int64())
	require.Equal(t, int64(200), feePaid.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 50)
	require.Nil(t, err)

	offer := math.NewInt(100)
	fee := math.LegacyNewDecWithPrec(2, 2) // 2%

	acc, _ := sdk.AccAddressFromBech32(keepertest.Carol)
	options := types.TradeOptions{
		GivenAmount:      offer,
		CoinSource:       acc,
		CoinTarget:       acc,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amount, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	amountUsed, amountReceived, feePaid, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amount))
	require.NoError(t, err)

	require.Equal(t, offer, amountUsed)
	require.Equal(t, int64(98), amountReceived.Int64())
	require.Equal(t, int64(2), feePaid.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(40000), pair.VirtualOther)

	maximum := k.CalculateSingleMaximumTradableAmount(ctx, utils.BaseCurrency, "ukusd", nil, false)
	require.NotNil(t, maximum)

	receivedAmount := k.ConstantProductTrade(ctx, utils.BaseCurrency, "ukusd", (*maximum).ToLegacyDec()).RoundInt()
	liqSum, _ := k.GetLiquiditySum(ctx, "ukusd")
	require.Equal(t, receivedAmount, liqSum)

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(40_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	offer := math.NewInt(10_000)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Carol)
	options := types.TradeOptions{
		CoinSource:       acc,
		CoinTarget:       acc,
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: dexkeeper.FlatPrice{},
		AllowIncomplete:  false,
	}

	amountUsed, amountReceived, feePaid, _, err := k.ExecuteTrade(ctx, ctx.EventManager(), options)

	require.NoError(t, err)
	require.Equal(t, int64(9_990), amountReceived.Int64())
	require.Equal(t, int64(10_000), amountUsed.Int64())
	require.Equal(t, int64(10), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestSingleTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(40_000), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Carol)

	options := types.TradeOptions{
		CoinSource:       acc,
		CoinTarget:       acc,
		GivenAmount:      math.NewInt(125_000),
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  false,
	}

	amountUsed, amountReceived, _, feePaid, err := k.ExecuteTrade(ctx, ctx.EventManager(), options)

	require.NoError(t, err)
	require.Equal(t, int64(9_990), amountReceived.Int64())
	require.Equal(t, int64(125_000), amountUsed.Int64())
	require.Equal(t, int64(10), feePaid.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.Nil(t, err)

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(2_000_000), liq)
	liq, found = k.GetLiquiditySum(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.NewInt(2_000_000), liq)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, keepertest.PowDec(18), pair.VirtualBase)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	res, err := msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.Nil(t, err)
	require.Equal(t, int64(95143), res.AmountReceived)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)

	coins := k.BankKeeper.SpendableCoins(ctx, addr)
	coinBase := getCoin(coins, utils.BaseCurrency)
	require.Equal(t, "99999000000", coinBase.Amount.String())

	coinKUSD := getCoin(coins, "ukusd")
	expected := 100000000000 + res.AmountReceived
	require.Equal(t, expected, coinKUSD.Amount.Int64())

	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(1))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(1))
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.NoError(t, err)
	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(2000000), liq)
}

func TestTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(1))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", keepertest.Pow(1))
	require.NoError(t, err)

	_, _ = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(2),
	})

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, liq, math.NewInt(6_000_000))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(1))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", keepertest.Pow(1))
	require.NoError(t, err)

	_, _ = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(2_000_000), liq)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})
	require.NoError(t, err)

	pair, _ := k.GetLiquidityPair(ctx, "ukusd")
	require.Equal(t, math.LegacyNewDec(18000000), pair.VirtualBase)
	require.Equal(t, math.LegacyNewDec(0), pair.VirtualOther)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade6(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    keepertest.PowInt64String(1),
	})

	require.NoError(t, err)

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	require.Equal(t, 1, len(coins))
	require.Equal(t, int64(48), coins[0].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000)
	require.NoError(t, err)

	_, _ = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "2000",
	})

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.True(t, pair.VirtualBase.GTE(math.LegacyZeroDec()))
	require.True(t, pair.VirtualOther.GTE(math.LegacyZeroDec()))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000)
	require.NoError(t, err)

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
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
}

func TestTrade9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 50000)
	require.NoError(t, err)

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
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
}

func TestTrade11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NoError(t, err)

	counter := 0
	for range k.GetLiquidityEntries(ctx, "ukusd") {
		counter++
	}

	require.Equal(t, 2, counter)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade12(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, "ukusd", 25_000)
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NoError(t, err)

	counter := 0
	for range k.GetLiquidityEntries(ctx, "ukusd") {
		counter++
	}

	require.Equal(t, 3, counter)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	var response *types.MsgTradeResponse
	response, err = msg.Trade(ctx, &types.MsgTrade{
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
}

func TestTrade15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyZeroDec(), pair.VirtualOther)

	var response *types.MsgTradeResponse
	response, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
	})

	require.NotNil(t, response)
	require.NoError(t, err)

	price := float64(response.AmountUsed) / float64(response.AmountReceived)
	require.Equal(t, price, 14.01541695865452)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(4000), pair.VirtualOther)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "50000",
	})
	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(4000), pair.VirtualOther)

	response, err := msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "12500",
	})

	require.NoError(t, err)
	// AmountReceived is not 1000 because of fee
	require.Equal(t, int64(999), response.AmountReceived)
	require.Equal(t, int64(12_500), response.AmountUsed)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 5_000)
	require.NoError(t, err)

	pair, found := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, found)
	require.Equal(t, math.LegacyNewDec(40000), pair.VirtualOther)

	response, err := msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "125000",
	})

	require.NoError(t, err)
	// AmountReceived is not 1000 because of fee
	require.Equal(t, int64(9990), response.AmountReceived)
	require.Equal(t, int64(125000), response.AmountUsed)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade19(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:         keepertest.Carol,
		DenomFrom:       utils.BaseCurrency,
		DenomTo:         "ukusd",
		Amount:          "2600",
		MaxPrice:        "10",
		AllowIncomplete: false,
	})

	require.Error(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:         keepertest.Carol,
		DenomFrom:       utils.BaseCurrency,
		DenomTo:         "ukusd",
		Amount:          "2600",
		MaxPrice:        "10.20",
		AllowIncomplete: false,
	})

	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade20(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 50_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 25_000)
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "2600",
		MaxPrice:  "10.10",
	})

	require.Error(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade22(t *testing.T) {
	amountReceived1 := testSmallDenomTrade(t, 20_000)
	amountReceived2 := testSmallDenomTrade(t, 1000)
	require.Greater(t, amountReceived1, amountReceived2)
}

func TestTrade23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.NoError(t, err)

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1000",
		MaxPrice:  "",
	})
	require.NoError(t, err)

	price2, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")

	require.NoError(t, err)
	require.True(t, price1.LT(price2))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func testSmallDenomTrade(t *testing.T, amount int64) int64 {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 25_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount)
	require.NoError(t, err)

	var response *types.MsgTradeResponse
	response, err = msg.Trade(ctx, &types.MsgTrade{
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

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)

	liqOtherSum1, _ := k.GetLiquiditySum(ctx, "ukusd")

	_, err = msg.Trade(ctx, &types.MsgTrade{
		Creator:   keepertest.Carol,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1000",
		MaxPrice:  "",
	})
	require.NoError(t, err)

	liqOtherSum2, _ := k.GetLiquiditySum(ctx, "ukusd")

	require.Equal(t, math.NewInt(1000), liqOtherSum2.Sub(liqOtherSum1))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade25(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount := int64(10_000)
	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, amount)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount)
	require.NoError(t, err)

	pair, ok := k.GetLiquidityPair(ctx, "ukusd")
	require.True(t, ok)

	A := k.GetFullLiquidityOther(ctx, "ukusd")
	B := k.GetFullLiquidityBase(ctx, "ukusd")
	b := B.Sub(pair.VirtualBase)
	c := pair.VirtualBase

	var maximum1 *math.LegacyDec
	if c.GT(math.LegacyZeroDec()) {
		m := A.Mul(b.Add(c)).Quo(c).Sub(A)
		maximum1 = &m
	}

	maximum2 := k.CalculateSingleMaximumTradableAmount(ctx, "ukusd", utils.BaseCurrency, nil, false)
	require.NotNil(t, maximum2)
	require.Equal(t, (*maximum1).TruncateInt(), *maximum2)

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
}

func TestTrade26(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount := int64(10_000)
	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, amount)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", amount)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", amount)
	require.NoError(t, err)

	var maximum1 *math.Int
	maximum1 = k.CalculateSingleMaximumTradableAmount(ctx, utils.BaseCurrency, "ukusd", nil, false)
	maximum1 = k.CalculateSingleMaximumTradableAmount(ctx, "uwusdc", utils.BaseCurrency, maximum1, false)

	var maximum2 *math.Int
	maximum2 = k.CalculateMaximumTradableAmount(ctx, "uwusdc", "ukusd", false)
	require.NotNil(t, maximum2)
	require.Equal(t, maximum1, maximum2)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade27(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000)
	require.NoError(t, err)
}

func TestTrade28(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade30(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100_000)
	require.NoError(t, err)

	maxPrice := math.LegacyNewDecWithPrec(105, 1)
	res, err := msg.Trade(ctx, &types.MsgTrade{
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
}

func TestTrade31(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10)
	require.NoError(t, err)

	maximumTradableAmount := k.CalculateMaximumTradableAmount(ctx, "ukusd", utils.BaseCurrency, false)
	fee := math.LegacyZeroDec()

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      *maximumTradableAmount,
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
	}

	_, amountReceivedNet, _, err := k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToBase(fee))
	require.NoError(t, err)
	_, _, _, err = k.ExecuteTradeStep(ctx, ctx.EventManager(), options.TradeToTarget(fee, amountReceivedNet))
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000)
	require.NoError(t, err)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      math.NewInt(1000),
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.FlatPrice{},
		AllowIncomplete:  true,
	}

	_, _, _, _, err = k.ExecuteTrade(ctx, ctx.EventManager(), options)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade33(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000)
	require.NoError(t, err)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      math.NewInt(1000),
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
	}

	_, _, _, _, err = k.ExecuteTrade(ctx, ctx.EventManager(), options)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10)
	require.NoError(t, err)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      math.NewInt(10000),
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeDenomStart:  utils.BaseCurrency,
		TradeDenomEnd:    "uwusdc",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
	}

	_, _, _, _, err = k.ExecuteTrade(ctx, ctx.EventManager(), options)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func TestTrade35(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 10_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 10_000)
	require.NoError(t, err)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		GivenAmount:      math.NewInt(1000),
		CoinSource:       addr,
		CoinTarget:       addr,
		TradeDenomStart:  "uwusdc",
		TradeDenomEnd:    "ukusd",
		TradeCalculation: dexkeeper.ConstantProduct{},
		AllowIncomplete:  true,
	}

	_, _, _, _, err = k.ExecuteTrade(ctx, ctx.EventManager(), options)
	require.NoError(t, err)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, tradePoolEmpty(ctx, k))
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liqSum, _ := k.GetLiquiditySum(ctx, denom)
		funds := coins.AmountOf(denom)

		diff := liqSum.Sub(funds).Abs()
		if diff.GT(math.NewInt(0)) {
			fmt.Println(denom)
			fmt.Println(fmt.Sprintf("liq sum: %v", liqSum.String()))
			fmt.Println(fmt.Sprintf("funds: %v", funds.String()))
			fmt.Println(fmt.Sprintf("diff: %v", diff.String()))

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
