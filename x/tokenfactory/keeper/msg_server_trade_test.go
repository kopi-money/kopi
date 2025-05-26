package keeper_test

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/trading"
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/constants"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestTrade1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1_000_000", constants.KUSD, "1_000_000", "0.01", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_000_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(1_000_000), pool.KCoinAmount.Int64())

	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10_000", "", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)

	require.Equal(t, 10_000, amountGivenGross)
	require.Equal(t, 9801, amountReceivedNet)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_010_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(990_150), pool.KCoinAmount.Int64())

	amountGivenGross, _ = strconv.Atoi(response.AmountGivenGross)
	amountReceivedNet, _ = strconv.Atoi(response.AmountReceivedNet)

	paidPrice1 := float64(amountGivenGross) / float64(amountReceivedNet)
	maxPriceString := fmt.Sprintf("%.8f", paidPrice1)

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10_000", maxPriceString, "", false)
	require.ErrorIs(t, err, trading.ErrMarketPriceTooHigh)
}

func TestTrade2(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.01", 10))

	response, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.ParseFloat(response.AmountReceivedGross, 64)
	amountReceivedNet, _ := strconv.ParseFloat(response.AmountReceivedNet, 64)

	paidPrice1 := amountGivenGross / amountReceivedNet
	maxPriceString := fmt.Sprintf("%.8f", paidPrice1)
	_, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, "", false)
	require.ErrorIs(t, err, trading.ErrMarketPriceTooHigh)

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, "", false)
	require.ErrorIs(t, err, trading.ErrMarketPriceTooHigh)
}

func TestTrade3(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	poolAmountFac := "400000000"
	poolAmountKCoin := "100000000"
	tradeAmount := "1000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, poolAmountFac))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, poolAmountFac, constants.KUSD, poolAmountKCoin, "0.01", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, poolAmountFac, pool.FactoryDenomAmount.String())
	require.Equal(t, poolAmountKCoin, pool.KCoinAmount.String())

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, poolAmountFac))

	_, err = k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	response1, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, "", "", true)
	require.NoError(t, err)

	response2, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, "", "", true)
	require.NoError(t, err)

	price1, _ := strconv.ParseFloat(response1.Price, 64)
	price2, _ := strconv.ParseFloat(response2.Price, 64)

	require.True(t, price2 > price1)
}

func TestTrade4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.01", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, "10000", "", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountGivenNet, _ := strconv.Atoi(response.AmountGivenNet)
	amountReceivedGross, _ := strconv.Atoi(response.AmountReceivedGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)
	feePool, _ := strconv.Atoi(response.FeePool)
	feeReserve, _ := strconv.Atoi(response.FeeReserve)

	require.Equal(t, 10000, amountGivenGross)
	require.Equal(t, 9900, amountGivenNet)
	require.Equal(t, 9802, amountReceivedGross)
	require.Equal(t, 9802, amountReceivedNet)
	require.Equal(t, 50, feePool)
	require.Equal(t, 50, feeReserve)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_009_950), pool.KCoinAmount.Int64())
	require.Equal(t, int64(990_198), pool.FactoryDenomAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceKCoin2-poolBalanceKCoin1, int64(amountGivenNet+feePool))
	require.Equal(t, poolBalanceFactory1-poolBalanceFactory2, int64(amountReceivedNet))
}

func TestTrade5(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.01", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountGivenNet, _ := strconv.Atoi(response.AmountGivenNet)
	amountReceivedGross, _ := strconv.Atoi(response.AmountReceivedGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)
	feePool, _ := strconv.Atoi(response.FeePool)
	feeReserve, _ := strconv.Atoi(response.FeeReserve)

	require.Equal(t, amountReceivedNet, amountReceivedGross-feeReserve-feePool)
	require.Equal(t, 10_000, amountGivenGross)
	require.Equal(t, 10_000, amountGivenNet)
	require.Equal(t, 9_900, amountReceivedGross)
	require.Equal(t, 9_801, amountReceivedNet)
	require.Equal(t, 49, feeReserve)
	require.Equal(t, 50, feePool)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_010_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(990_150), pool.KCoinAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceFactory2-poolBalanceFactory1, int64(amountGivenGross))
	require.Equal(t, poolBalanceKCoin1-poolBalanceKCoin2, int64(amountReceivedNet+feeReserve))
}

func TestTrade6(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.01", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, "1000", "", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountGivenNet, _ := strconv.Atoi(response.AmountGivenNet)
	amountReceivedGross, _ := strconv.Atoi(response.AmountReceivedGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)
	feePool, _ := strconv.Atoi(response.FeePool)
	feeReserve, _ := strconv.Atoi(response.FeeReserve)

	require.Equal(t, 1012, amountGivenGross)
	require.Equal(t, 1001, amountGivenNet)
	require.Equal(t, 1000, amountReceivedGross)
	require.Equal(t, 1000, amountReceivedNet)
	require.Equal(t, 6, feePool)
	require.Equal(t, 5, feeReserve)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_001_007), pool.KCoinAmount.Int64())
	require.Equal(t, int64(999_000), pool.FactoryDenomAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceFactory1-poolBalanceFactory2, int64(amountReceivedNet))
	require.Equal(t, poolBalanceKCoin2-poolBalanceKCoin1, int64(amountGivenNet+feePool))
}

func TestTrade7aa(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	res2, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade7ab(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res1.AmountGiven))

	res2, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade7ba(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, res1.AmountGiven))

	res2, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade7bb(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "5_000_000"))

	res2, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade7bbb(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res1.AmountGiven))

	res2, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade8(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res1.AmountGiven))

	res2, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade9(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100000000"   // 100
	liqFactory := "400000000" // 400
	tradeAmount := "1000000"  // 1

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, liqKCoin, pool.KCoinAmount.String())
	require.Equal(t, liqFactory, pool.FactoryDenomAmount.String())

	res2, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestTrade10(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	res1, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res1.AmountGiven))

	res2, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res1.Price, "", true)
	require.NoError(t, err)

	require.Equal(t, res1.AmountGiven, res2.AmountGivenGross)
	require.Equal(t, res1.AmountReceived, res2.AmountReceivedNet)
}

func TestMaxPrice1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	maxPrice, _ := math.LegacyNewDecFromStr("0.25")
	tradeAmount := math.NewInt(1)
	liqFrom, liqTo := pool.GetLiquidityAmounts("ukusd")

	tradeData := trading.TradeData{
		TradeAmount: tradeAmount,
		MaxPrice: &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		},
		LiqFrom:   liqFrom,
		LiqTo:     liqTo,
		Callbacks: trading.SellCallbacks(),
		Fee:       pool.PoolFee,
	}

	_, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	require.Error(t, err)
}

func TestMaxPrice2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         "100000000",
	})
	require.NoError(t, err)
	fmt.Println(res.Price)

	maxPrice, _ := math.LegacyNewDecFromStr("69000")
	tradeAmount := math.NewInt(1)
	liqFrom, liqTo := pool.GetLiquidityAmounts("ukusd")

	tradeData := trading.TradeData{
		TradeAmount: tradeAmount,
		MaxPrice: &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		},
		LiqFrom:   liqFrom,
		LiqTo:     liqTo,
		Callbacks: trading.SellCallbacks(),
		Fee:       pool.PoolFee,
	}

	_, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	require.Error(t, err)

	maxPrice, _ = math.LegacyNewDecFromStr("70000")
	tradeData.MaxPrice = &trading.MaxPriceData{
		MaxPrice:    maxPrice,
		FeeIncluded: true,
	}
	_, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	require.NoError(t, err)
}

func TestMaxPrice4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.01", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	maxPrice, _ := math.LegacyNewDecFromStr("69000")
	tradeAmount := math.NewInt(1)
	liqFrom, liqTo := pool.GetLiquidityAmounts("ukusd")

	tradeData := trading.TradeData{
		TradeAmount: tradeAmount,
		MaxPrice: &trading.MaxPriceData{
			MaxPrice:    maxPrice,
			FeeIncluded: true,
		},
		LiqFrom:   liqFrom,
		LiqTo:     liqTo,
		Callbacks: trading.SellCallbacks(),
		Fee:       pool.PoolFee,
	}

	_, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	require.Error(t, err)

	maxPrice, _ = math.LegacyNewDecFromStr("70000")
	tradeData.MaxPrice = &trading.MaxPriceData{
		MaxPrice:    maxPrice,
		FeeIncluded: true,
	}

	_, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	require.NoError(t, err)
}

func TestPoolChange1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "10_000", constants.KUSD, "10_000", "0.01", 10))

	factoryDenom, has := k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, int64(10_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(10_000), pool.KCoinAmount.Int64())

	tradeCtx := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.SellCallbacks(),
		TradeAmount:    math.NewInt(1000),
		Pool:           pool,
		DenomGiving:    factoryDenomHash,
		DenomReceiving: pool.KCoin,
		Creator:        keepertest.Alice,
		CPTrade:        trading.FlatTrade,
	}

	var res *types.MsgTradeResponse
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		res, err = k.Trade(tradeCtx, factoryDenom)
		return err
	}))

	require.Equal(t, "1000", res.AmountGivenGross)
	require.Equal(t, "1000", res.AmountGivenNet)
	require.Equal(t, "1000", res.AmountReceivedGross)
	require.Equal(t, "990", res.AmountReceivedNet)

	require.Equal(t, "10", res.Fee)
	require.Equal(t, "5", res.FeePool)
	require.Equal(t, "5", res.FeeReserve)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(11_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(9_005), pool.KCoinAmount.Int64())
}

func TestPoolChange2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100_000", constants.KUSD, "100_000", "0.01", 10))

	factoryDenom, has := k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, int64(100_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(100_000), pool.KCoinAmount.Int64())

	tradeCtx := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.SellCallbacks(),
		TradeAmount:    math.NewInt(10_000),
		Pool:           pool,
		DenomGiving:    pool.KCoin,
		DenomReceiving: factoryDenomHash,
		Creator:        keepertest.Alice,
		CPTrade:        trading.FlatTrade,
	}

	var res *types.MsgTradeResponse
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		res, err = k.Trade(tradeCtx, factoryDenom)
		return err
	}))

	require.Equal(t, "10000", res.AmountGivenGross)
	require.Equal(t, "9900", res.AmountGivenNet)
	require.Equal(t, "9900", res.AmountReceivedGross)
	require.Equal(t, "9900", res.AmountReceivedNet)

	require.Equal(t, "100", res.Fee)
	require.Equal(t, "50", res.FeePool)
	require.Equal(t, "50", res.FeeReserve)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(109_950), pool.KCoinAmount.Int64())
	require.Equal(t, int64(90_100), pool.FactoryDenomAmount.Int64())
}

func TestPoolChange3(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100_000", constants.KUSD, "100_000", "0.01", 10))

	factoryDenom, has := k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, int64(100_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(100_000), pool.KCoinAmount.Int64())

	tradeCtx := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.BuyCallbacks(),
		TradeAmount:    math.NewInt(10_000),
		Pool:           pool,
		DenomGiving:    factoryDenomHash,
		DenomReceiving: pool.KCoin,
		Creator:        keepertest.Alice,
		CPTrade:        trading.FlatTrade,
	}

	var res *types.MsgTradeResponse
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		res, err = k.Trade(tradeCtx, factoryDenom)
		return err
	}))

	// The amounts are calculated using the fees and are rounding differently. The unrounded amounts are equal
	require.Equal(t, "10102", res.AmountGivenGross)
	require.Equal(t, "10102", res.AmountGivenNet)
	require.Equal(t, "10101", res.AmountReceivedGross)
	require.Equal(t, "10000", res.AmountReceivedNet)

	require.Equal(t, "101", res.Fee)
	require.Equal(t, "51", res.FeePool)
	require.Equal(t, "50", res.FeeReserve)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(110_102), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(89_950), pool.KCoinAmount.Int64())
}

func TestPoolChange4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100_000", constants.KUSD, "100_000", "0.01", 10))

	factoryDenom, has := k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, int64(100_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(100_000), pool.KCoinAmount.Int64())

	tradeCtx := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.BuyCallbacks(),
		TradeAmount:    math.NewInt(10_000),
		Pool:           pool,
		DenomGiving:    pool.KCoin,
		DenomReceiving: factoryDenomHash,
		Creator:        keepertest.Alice,
		CPTrade:        trading.FlatTrade,
	}

	var res *types.MsgTradeResponse
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		res, err = k.Trade(tradeCtx, factoryDenom)
		return err
	}))

	require.Equal(t, "10102", res.AmountGivenGross)
	require.Equal(t, "10000", res.AmountGivenNet)
	require.Equal(t, "10000", res.AmountReceivedGross)
	require.Equal(t, "10000", res.AmountReceivedNet)

	require.Equal(t, "102", res.Fee)
	require.Equal(t, "51", res.FeePool)
	require.Equal(t, "51", res.FeeReserve)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(90_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(110_051), pool.KCoinAmount.Int64())
}
