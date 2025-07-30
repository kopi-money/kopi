package keeper_test

import (
	"context"
	"testing"

	"github.com/kopi-money/kopi/trading"

	"github.com/cosmos/cosmos-sdk/cache"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestMint1(t *testing.T) {
	k, dexMsg, dexK, _, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, constants.BaseCurrency, 100000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, constants.KUSD, 100000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, "uwusdc", 100000))
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, constants.KUSD, 10)

	zeroDec := math.LegacyZeroDec()
	tradeCtx := dextypes.TradeContext{
		Context:             ctx,
		CoinSource:          addr.String(),
		CoinTarget:          addr.String(),
		TradeAmount:         math.NewInt(5_000_000),
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		Fee:                 &zeroDec,
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.DexKeeper.ExecuteSell(tradeCtx)
		return err
	}))

	require.True(t, tradeResult.AmountGiven().IsPositive())

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, constants.KUSD)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CheckMint(innerCtx, constants.KUSD, maxMintAmount)
	}))

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")

	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, dexK))
}

func TestMint2(t *testing.T) {
	supply1 := mintScenario(t, 5000_000000)
	supply2 := mintScenario(t, 10000_000000)

	require.Equal(t, supply1, supply2)
}

func mintScenario(t *testing.T, buyAmount int64) int64 {
	k, dexMsg, dexK, _, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, constants.BaseCurrency, 100000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, constants.KUSD, 100000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, "uwusdc", 100000))
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, constants.KUSD, 10)

	tradeCtx := dextypes.TradeContext{
		CoinSource:          addr.String(),
		CoinTarget:          addr.String(),
		TradeAmount:         math.NewInt(buyAmount),
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.KUSD,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.DexKeeper.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	require.True(t, tradeResult.AmountGiven().IsPositive())

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, constants.KUSD)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CheckMint(innerCtx, constants.KUSD, maxMintAmount)
	}))

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")

	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, dexK))

	return k.BankKeeper.GetSupply(ctx, constants.KUSD).Amount.Int64()
}

func TestMint3(t *testing.T) {
	k, dexMsg, _, _, ctx := keepertest.SetupSwapMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, constants.BaseCurrency, 100000))
	require.NoError(t, keepertest.AddLiquidity(ctx, dexMsg, keepertest.Bob, "uwusdc", 100000))

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	parity1, _, err := k.DenomKeeper.CalculateParity(ctx, constants.KUSD)
	require.NoError(t, err)
	require.NotNil(t, parity1)

	tradeCtx := dextypes.TradeContext{
		CoinSource:          addr.String(),
		CoinTarget:          addr.String(),
		TradeAmount:         math.NewInt(100_000_000_000),
		TradeDenomGiving:    "uwusdc",
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		_, err = k.DexKeeper.ExecuteSell(tradeCtx)
		return err
	}))

	require.NoError(t, tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper))

	supply1 := k.BankKeeper.GetSupply(ctx, constants.KUSD).Amount

	parity2, _, err := k.DenomKeeper.CalculateParity(ctx, constants.KUSD)
	require.NoError(t, err)
	require.NotNil(t, parity2)

	require.Equal(t, parity1, parity2)

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, constants.KUSD)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CheckMint(innerCtx, constants.KUSD, maxMintAmount)
	}))

	supply2 := k.BankKeeper.GetSupply(ctx, constants.KUSD).Amount

	// supply has to be unchanged because uwusdt is used as reference after uwusdc "crashed"
	require.True(t, supply1.Equal(supply2))
}
