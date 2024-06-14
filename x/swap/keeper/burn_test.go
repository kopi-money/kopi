package keeper_test

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/cache"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/swap/keeper"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
)

func TestBurn1(t *testing.T) {
	k, msg, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)
	addLiquidity(ctx, k, t, "ukusd", 100000)
	addLiquidity(ctx, k, t, "uwusdc", 100000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "uwusdc", 10)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	reserveCoins := sdk.NewCoins(sdk.NewCoin("uwusdc", math.NewInt(10)))
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress()
	err := k.BankKeeper.SendCoins(ctx, addr, acc, reserveCoins)
	require.NoError(t, err)

	price1, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)

	_, err = keepertest.Trade(ctx, msg, &dextypes.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   "uwusdc",
		Amount:    "10000",
	})
	require.NoError(t, err)

	price2, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	dexKeeper := k.DexKeeper.(dexkeeper.Keeper)
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return dexKeeper.BeginBlockCheckReserve(innerCtx, innerCtx.EventManager(), innerCtx.BlockHeight())
	}))

	priceBase, err := k.DexKeeper.CalculatePrice(ctx, utils.BaseCurrency, "uwusdc")
	require.NoError(t, err)
	require.False(t, priceBase.IsNil())

	maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, "ukusd")
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.CheckBurn(innerCtx, innerCtx.EventManager(), "ukusd", maxBurnAmount)
	}))
	require.True(t, liquidityBalanced(ctx, dexK))

	price3, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price3.LT(price2))
}

func TestBurn2(t *testing.T) {
	k, msg, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 100000)
	addLiquidity(ctx, k, t, "ukusd", 100000)
	addLiquidity(ctx, k, t, "uwusdc", 100000)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	reserveCoins := sdk.NewCoins(sdk.NewCoin("uwusdc", math.NewInt(10)))
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress()
	err := k.BankKeeper.SendCoins(ctx, addr, acc, reserveCoins)
	require.NoError(t, err)

	price1, _ := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")

	_, err = keepertest.Trade(ctx, msg, &dextypes.MsgTrade{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   "uwusdc",
		Amount:    "10000",
	})
	require.NoError(t, err)

	price2, err := k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	priceBase, err := k.DexKeeper.CalculatePrice(ctx, utils.BaseCurrency, "uwusdc")
	require.NoError(t, err)
	require.False(t, priceBase.IsNil())

	for i := 0; i < 10; i++ {
		require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
			return k.Burn(innerCtx, innerCtx.EventManager())
		}))

		var price3 math.LegacyDec
		price3, err = k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
		require.NoError(t, err)
		require.True(t, price3.LT(price2))
	}

	require.True(t, liquidityBalanced(ctx, dexK))
}

func addLiquidity(ctx sdk.Context, k keeper.Keeper, t *testing.T, denom string, amount int64) {
	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.DexKeeper.AddLiquidity(innerCtx, innerCtx.EventManager(), addr, denom, math.NewInt(amount))
	}))

	liq := k.DexKeeper.GetLiquiditySum(ctx, denom)
	require.Equal(t, liq.Int64(), amount)
}

func addReserveFundsToDex(ctx sdk.Context, acc swaptypes.AccountKeeper, dex swaptypes.DexKeeper, bank swaptypes.BankKeeper, t *testing.T, denom string, amount int64) {
	reserveAcc := acc.GetModuleAccount(ctx, dextypes.PoolReserve)

	coin := sdk.NewCoin(denom, math.LegacyNewDec(amount*2).RoundInt())
	coins := sdk.NewCoins(coin)
	err := bank.MintCoins(ctx, swaptypes.ModuleName, coins)
	require.NoError(t, err)
	addr, err := sdk.AccAddressFromBech32(reserveAcc.GetAddress().String())
	require.NoError(t, err)

	mintAcc := acc.GetModuleAccount(ctx, swaptypes.ModuleName).GetAddress()
	err = bank.SendCoins(ctx, mintAcc, addr, coins)
	require.NoError(t, err)

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return dex.AddLiquidity(innerCtx, innerCtx.EventManager(), reserveAcc.GetAddress(), denom, math.LegacyNewDec(amount).RoundInt())
	}))
}

func TestBurn3(t *testing.T) {
	supply1 := burnScenario(t, 1000)
	supply2 := burnScenario(t, 1000000)

	require.Greater(t, supply2, supply1)
}

func burnScenario(t *testing.T, sellAmount int64) int64 {
	k, _, dexK, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	addLiquidity(ctx, k, t, utils.BaseCurrency, 10000)
	addLiquidity(ctx, k, t, "ukusd", 10000)
	addLiquidity(ctx, k, t, "uwusdc", 10000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "ukusd", 10)

	tradeCtx := dextypes.TradeContext{
		Context:         ctx,
		CoinSource:      addr.String(),
		CoinTarget:      addr.String(),
		GivenAmount:     math.NewInt(sellAmount),
		TradeDenomStart: "ukusd",
		TradeDenomEnd:   "uwusdc",
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
	require.True(t, price1.GT(math.LegacyOneDec()))

	maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, "ukusd")

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		return k.CheckBurn(innerCtx, innerCtx.EventManager(), "ukusd", maxBurnAmount)
	}))
	_, err = k.DexKeeper.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.True(t, liquidityBalanced(ctx, dexK))

	return k.BankKeeper.GetSupply(ctx, "ukopi").Amount.Int64()
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liqSum := k.GetLiquiditySum(ctx, denom)
		funds := coins.AmountOf(denom)

		diff := liqSum.Sub(funds).Abs()
		if diff.GT(math.NewInt(1)) {
			fmt.Println(denom)
			fmt.Println(fmt.Sprintf("liq sum: %v", liqSum.String()))
			fmt.Println(fmt.Sprintf("funds: %v", funds.String()))
			fmt.Println(fmt.Sprintf("diff: %v", diff.String()))

			return false
		}
	}

	return true
}
