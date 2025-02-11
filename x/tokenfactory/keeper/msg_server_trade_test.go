package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	"github.com/kopi-money/kopi/x/tokenfactory/keeper"
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/constants"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestTrade1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_000_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(1_000_000), pool.KCoinAmount.Int64())

	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)

	require.Equal(t, 10000, amountGivenGross)
	require.Equal(t, 8901, amountReceivedNet)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1010000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(992089), pool.KCoinAmount.Int64())

	amountGivenGross, _ = strconv.Atoi(response.AmountGivenGross)
	amountReceivedNet, _ = strconv.Atoi(response.AmountReceivedNet)

	paidPrice1 := float64(amountGivenGross) / float64(amountReceivedNet)
	maxPriceString := fmt.Sprintf("%.8f", paidPrice1)
	response, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, false)
	require.ErrorIs(t, err, types.ErrMarketPriceTooHigh)
	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, true)
	require.ErrorIs(t, err, types.ErrMarketPriceTooHigh)
}

func TestTrade2(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	response, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.ParseFloat(response.AmountReceivedGross, 64)
	amountReceivedNet, _ := strconv.ParseFloat(response.AmountReceivedNet, 64)

	paidPrice1 := amountGivenGross / amountReceivedNet
	maxPriceString := fmt.Sprintf("%.8f", paidPrice1)
	response, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, false)
	require.ErrorIs(t, err, types.ErrMarketPriceTooHigh)

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", maxPriceString, true)
	require.ErrorIs(t, err, types.ErrMarketPriceTooHigh)
}

func TestTrade3(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	poolAmountFac := "400000000"
	poolAmountKCoin := "100000000"
	tradeAmount := "1000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, poolAmountFac))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, poolAmountFac, constants.KUSD, poolAmountKCoin, "0.1", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, poolAmountFac, pool.FactoryDenomAmount.String())
	require.Equal(t, poolAmountKCoin, pool.KCoinAmount.String())

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, poolAmountFac))

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	response1, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res.Price, true)
	require.NoError(t, err)

	response2, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, response1.Price, true)
	require.NoError(t, err)

	price1, _ := strconv.ParseFloat(response1.Price, 64)
	price2, _ := strconv.ParseFloat(response2.Price, 64)
	require.True(t, price1 > price2)
}

func TestTrade4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, "10000", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountGivenNet, _ := strconv.Atoi(response.AmountGivenNet)
	amountReceivedGross, _ := strconv.Atoi(response.AmountReceivedGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)
	feePool, _ := strconv.Atoi(response.FeePool)
	feeReserve, _ := strconv.Atoi(response.FeeReserve)

	require.Equal(t, 10000, amountGivenGross)
	require.Equal(t, 8990, amountGivenNet)
	require.Equal(t, 8909, amountReceivedGross)
	require.Equal(t, 8909, amountReceivedNet)
	require.Equal(t, 1000, feePool)
	require.Equal(t, 10, feeReserve)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_009_990), pool.KCoinAmount.Int64())
	require.Equal(t, int64(991_091), pool.FactoryDenomAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceKCoin2-poolBalanceKCoin1, int64(amountGivenNet+feePool))
	require.Equal(t, poolBalanceFactory1-poolBalanceFactory2, int64(amountReceivedNet))
}

func TestTrade5(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "", true)
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
	require.Equal(t, 8_901, amountReceivedNet)
	require.Equal(t, 9, feeReserve)
	require.Equal(t, 990, feePool)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_010_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(992_089), pool.KCoinAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceFactory2-poolBalanceFactory1, int64(amountGivenGross))
	require.Equal(t, poolBalanceKCoin1-poolBalanceKCoin2, int64(amountReceivedNet+feeReserve))
}

func TestTrade6(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	poolBalance1 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory1 := poolBalance1.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin1 := poolBalance1.AmountOf(constants.KUSD).Int64()

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "10000"))
	response, err := keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, "1000", "", true)
	require.NoError(t, err)

	amountGivenGross, _ := strconv.Atoi(response.AmountGivenGross)
	amountGivenNet, _ := strconv.Atoi(response.AmountGivenNet)
	amountReceivedGross, _ := strconv.Atoi(response.AmountReceivedGross)
	amountReceivedNet, _ := strconv.Atoi(response.AmountReceivedNet)
	feePool, _ := strconv.Atoi(response.FeePool)
	feeReserve, _ := strconv.Atoi(response.FeeReserve)

	require.Equal(t, 1000, amountReceivedNet)
	require.Equal(t, 1000, amountReceivedGross)
	require.Equal(t, 1001, amountGivenNet)
	require.Equal(t, 1102, amountGivenGross)
	require.Equal(t, 100, feePool)
	require.Equal(t, 1, feeReserve)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1001101), pool.KCoinAmount.Int64())
	require.Equal(t, int64(999000), pool.FactoryDenomAmount.Int64())

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

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade7ab(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res.AmountGiven))

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade7ba(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, res.AmountGiven))

	_, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade7bb(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res.AmountGiven))

	_, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade8(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res.AmountGiven))

	_, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade9(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100000000"   // 100
	liqFactory := "400000000" // 400
	tradeAmount := "1000000"  // 1

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	require.Equal(t, liqKCoin, pool.KCoinAmount.String())
	require.Equal(t, liqFactory, pool.FactoryDenomAmount.String())

	_, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, constants.KUSD, factoryDenomHash, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestTrade10(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"
	tradeAmount := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.1", 10))

	res, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    factoryDenomHash,
		DenomReceiving: constants.KUSD,
		Amount:         tradeAmount,
	})
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, res.AmountGiven))

	_, err = keepertest.FactoryDenomBuy(ctx, msgServer, keepertest.Bob, factoryDenomHash, factoryDenomHash, constants.KUSD, tradeAmount, res.Price, true)
	require.NoError(t, err)
}

func TestMaxPrice1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "100_000000"
	liqFactory := "400_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.0", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	tradeData := keeper.NewTradeData(factoryDenomHash, keepertest.Alice, factoryDenomHash, constants.KUSD, "0.25", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.DecreaseMaxPrice, constant_product.CalculateMaximumGiving)
	require.Error(t, err)
}

func TestMaxPrice2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.0", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Amount:         "100000000",
	})
	require.NoError(t, err)
	fmt.Println(res.Price)

	tradeData := keeper.NewTradeData(factoryDenomHash, keepertest.Alice, constants.KUSD, factoryDenomHash, "69000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.DecreaseMaxPrice, constant_product.CalculateMaximumGiving)
	require.Error(t, err)

	tradeData = keeper.NewTradeData(factoryDenomHash, keepertest.Alice, constants.KUSD, factoryDenomHash, "70000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.DecreaseMaxPrice, constant_product.CalculateMaximumGiving)
	require.NoError(t, err)
}

func TestMaxPrice3(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.0", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	tradeData := keeper.NewTradeData(factoryDenomHash, keepertest.Alice, factoryDenomHash, constants.KUSD, "69000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.IncreaseMaxPrice, constant_product.CalculateMaximumGiving)
	require.NoError(t, err)

	tradeData = keeper.NewTradeData(factoryDenomHash, keepertest.Alice, factoryDenomHash, constants.KUSD, "70000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.IncreaseMaxPrice, constant_product.CalculateMaximumGiving)
	require.Error(t, err)
}

func TestMaxPrice4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.0", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	tradeData := keeper.NewTradeData(factoryDenomHash, keepertest.Alice, constants.KUSD, factoryDenomHash, "69000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.DecreaseMaxPrice, constant_product.CalculateMaximumReceiving)
	require.Error(t, err)

	tradeData = keeper.NewTradeData(factoryDenomHash, keepertest.Alice, constants.KUSD, factoryDenomHash, "70000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.DecreaseMaxPrice, constant_product.CalculateMaximumReceiving)
	require.NoError(t, err)
}

func TestMaxPrice5(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	liqKCoin := "69000_000000"
	liqFactory := "1_000000"

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, liqFactory))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, liqFactory, constants.KUSD, liqKCoin, "0.0", 10))

	pool, has := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.True(t, has)

	tradeData := keeper.NewTradeData(factoryDenomHash, keepertest.Alice, factoryDenomHash, constants.KUSD, "69000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.IncreaseMaxPrice, constant_product.CalculateMaximumReceiving)
	require.NoError(t, err)

	tradeData = keeper.NewTradeData(factoryDenomHash, keepertest.Alice, factoryDenomHash, constants.KUSD, "70000", "1", false)
	_, _, err = k.HandleMaxPrice(ctx, tradeData, pool, math.NewInt(1), keeper.IncreaseMaxPrice, constant_product.CalculateMaximumReceiving)
	require.Error(t, err)
}
