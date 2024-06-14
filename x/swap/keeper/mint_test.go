package keeper_test

import (
	"github.com/kopi-money/kopi/cache"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
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

	tradeCtx := dextypes.TradeContext{
		Context:         ctx,
		CoinSource:      addr.String(),
		CoinTarget:      addr.String(),
		GivenAmount:     math.NewInt(5000),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   "ukusd",
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	var amountUsed math.Int
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		amountUsed, _, _, _, _, err = k.DexKeeper.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, amountUsed.GT(math.ZeroInt()))

	price1, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.CheckMint(innerCtx, innerCtx.EventManager(), "ukusd", maxMintAmount)
	}))

	price2, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")

	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.True(t, liquidityBalanced(ctx, dexK))
}

func TestMint2(t *testing.T) {
	supply1 := mintScenario(t, 5000)
	supply2 := mintScenario(t, 10000)

	require.Less(t, supply1, supply2)
}

func mintScenario(t *testing.T, buyAmount int64) int64 {
	k, _, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)
	addLiquidity(ctx, k, t, "ukusd", 100000)
	addLiquidity(ctx, k, t, "uwusdc", 100000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "ukusd", 10)

	tradeCtx := dextypes.TradeContext{
		CoinSource:      addr.String(),
		CoinTarget:      addr.String(),
		GivenAmount:     math.NewInt(buyAmount),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   "ukusd",
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	var amountUsed math.Int
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		amountUsed, _, _, _, _, err = k.DexKeeper.ExecuteTrade(tradeCtx)
		return err
	}))

	require.True(t, amountUsed.GT(math.ZeroInt()))

	price1, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price1.LT(math.LegacyOneDec()))

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.CheckMint(innerCtx, innerCtx.EventManager(), "ukusd", maxMintAmount)
	}))

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

	tradeCtx := dextypes.TradeContext{
		CoinSource:      addr.String(),
		CoinTarget:      addr.String(),
		GivenAmount:     math.NewInt(100_000_000_000),
		TradeDenomStart: "uwusdc",
		TradeDenomEnd:   utils.BaseCurrency,
		AllowIncomplete: true,
		MaxPrice:        nil,
	}

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, _, _, _, err = k.DexKeeper.ExecuteTrade(tradeCtx)
		return err
	}))

	supply1 := k.BankKeeper.GetSupply(ctx, "ukusd").Amount

	maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, "ukusd")
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.CheckMint(innerCtx, innerCtx.EventManager(), "ukusd", maxMintAmount)
	}))

	supply2 := k.BankKeeper.GetSupply(ctx, "ukusd").Amount

	// supply has to be unchanged because uwusdt is used as reference after uwusdc "crashed"
	require.True(t, supply1.Equal(supply2))
}
