package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestMint1(t *testing.T) {
	k, _, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)
	addLiquidity(ctx, k, t, "ukusd", 100000)
	addLiquidity(ctx, k, t, "uwusdc", 100000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "ukusd", 10)

	tradeOptions := dextypes.TradeOptions{
		CoinSource:      addr,
		CoinTarget:      addr,
		GivenAmount:     math.NewInt(1000),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   "ukusd",
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	amountUsed, _, _, _, err := k.DexKeeper.ExecuteTrade(ctx, ctx.EventManager(), tradeOptions)
	require.NoError(t, err)
	require.True(t, amountUsed.GT(math.ZeroInt()))

	price1, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, k.CheckMint(ctx, ctx.EventManager(), "ukusd", maxMintAmount))

	price2, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")

	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, dexK))
}

func TestMint2(t *testing.T) {
	supply1 := mintScenario(t, 1000)
	supply2 := mintScenario(t, 10000)

	require.Greater(t, supply1, supply2)
}

func mintScenario(t *testing.T, buyAmount int64) int64 {
	k, _, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)
	addLiquidity(ctx, k, t, "ukusd", 100000)
	addLiquidity(ctx, k, t, "uwusdc", 100000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "ukusd", 10)

	tradeOptions := dextypes.TradeOptions{
		CoinSource:      addr,
		CoinTarget:      addr,
		GivenAmount:     math.NewInt(buyAmount),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   "ukusd",
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	amountUsed, _, _, _, err := k.DexKeeper.ExecuteTrade(ctx, ctx.EventManager(), tradeOptions)
	require.NoError(t, err)
	require.True(t, amountUsed.GT(math.ZeroInt()))

	price1, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, k.CheckMint(ctx, ctx.EventManager(), "ukusd", maxMintAmount))

	price2, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")

	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, dexK))

	return k.BankKeeper.GetSupply(ctx, "ukusd").Amount.Int64()
}

func TestMint3(t *testing.T) {
	k, _, _, ctx := keepertest.SetupSwapMsgServer(t)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	tradeOptions := dextypes.TradeOptions{
		CoinSource:      addr,
		CoinTarget:      addr,
		GivenAmount:     math.NewInt(100_000_000_000),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   utils.BaseCurrency,
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	_, _, _, _, err = k.DexKeeper.ExecuteTrade(ctx, ctx.EventManager(), tradeOptions)
	require.NoError(t, err)

	supply1 := k.BankKeeper.GetSupply(ctx, "ukusd").Amount

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, k.CheckMint(ctx, ctx.EventManager(), "ukusd", maxMintAmount))

	supply2 := k.BankKeeper.GetSupply(ctx, "ukusd").Amount

	// supply has to be unchanged because uwusdt is used as reference after uwusdc "crashed"
	require.True(t, supply1.Equal(supply2))
}
