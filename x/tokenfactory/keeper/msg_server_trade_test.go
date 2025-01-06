package keeper_test

import (
	"fmt"
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

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.1", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1000000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(1000000), pool.KCoinAmount.Int64())

	response, err := keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "1000", "1.11", true)
	require.ErrorIs(t, err, types.ErrMarketPriceTooHigh)

	response, err = keepertest.FactoryDenomSell(ctx, msgServer, keepertest.Alice, factoryDenomHash, factoryDenomHash, constants.KUSD, "10000", "1.12", true)
	require.NoError(t, err, types.ErrEmptyTrade)

	amountGivenGross, _ := strconv.ParseFloat(response.AmountReceivedGross, 64)
	amountReceivedNet, _ := strconv.ParseFloat(response.AmountReceivedNet, 64)

	require.True(t, amountGivenGross < 10000)
	require.True(t, amountReceivedNet < 8910)
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
	require.Equal(t, 8901, amountReceivedGross)
	require.Equal(t, 8901, amountReceivedNet)
	require.Equal(t, 1000, feePool)
	require.Equal(t, 10, feeReserve)

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1_009_990), pool.KCoinAmount.Int64())
	require.Equal(t, int64(991_099), pool.FactoryDenomAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceKCoin2-poolBalanceKCoin1, int64(amountGivenGross))
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
	require.Equal(t, poolBalanceKCoin1-poolBalanceKCoin2, int64(amountReceivedNet))
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
	require.Equal(t, int64(1001202), pool.KCoinAmount.Int64())
	require.Equal(t, int64(999000), pool.FactoryDenomAmount.Int64())

	poolBalance2 := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	poolBalanceFactory2 := poolBalance2.AmountOf(factoryDenomHash).Int64()
	poolBalanceKCoin2 := poolBalance2.AmountOf(constants.KUSD).Int64()

	require.Equal(t, poolBalanceFactory1-poolBalanceFactory2, int64(amountReceivedNet))
	require.Equal(t, poolBalanceKCoin2-poolBalanceKCoin1, int64(amountGivenGross))
}
