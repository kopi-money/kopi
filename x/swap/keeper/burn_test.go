package keeper_test

import (
	"context"
	"fmt"
	gomath "math"
	"testing"

	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/trading"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	swaptypes "github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
)

func TestBurn1(t *testing.T) {
	k, msg, dexK, reserveK, ctx := keepertest.SetupSwapMsgServer(t)

	keepertest.AddFunds(ctx, t, dexK.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, dexK.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, dexK.BankKeeper, "uwusdc", keepertest.Alice, 1_000000_000000)

	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, constants.BaseCurrency, 4_000000_000000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, constants.KUSD, 1_000000_000000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, "uwusdc", 1_000000_000000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, "uwusdc", 10)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	reserveCoins := sdk.NewCoins(sdk.NewCoin("uwusdc", math.NewInt(10)))
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress()
	err := k.BankKeeper.SendCoins(ctx, addr, acc, reserveCoins)
	require.NoError(t, err)

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)

	_, err = keepertest.Sell(ctx, msg, &dextypes.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "1_00000_000000",
	})
	require.NoError(t, err)

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return reserveK.BeginBlockCheckReserve(innerCtx)
	}))

	priceBase, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)
	require.False(t, priceBase.IsNil())

	maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, constants.KUSD)
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CheckBurn(innerCtx, constants.KUSD, maxBurnAmount)
	}))
	require.True(t, liquidityBalanced(ctx, dexK))

	price3, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)

	price2F, _ := price2.Float64()
	price3F, _ := price3.Float64()

	require.Less(t, price3F, price2F)
}

func TestBurn2(t *testing.T) {
	k, msg, dexK, _, ctx := keepertest.SetupSwapMsgServer(t)

	keepertest.AddFunds(ctx, t, dexK.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_000000_000000)
	keepertest.AddFunds(ctx, t, dexK.BankKeeper, constants.KUSD, keepertest.Alice, 1_000000_000000)
	keepertest.AddFunds(ctx, t, dexK.BankKeeper, "uwusdc", keepertest.Alice, 1_000000_000000)

	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, constants.BaseCurrency, 4_000000_000000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, constants.KUSD, 1_000000_000000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, keepertest.Alice, "uwusdc", 1_000000_000000)

	addr, _ := sdk.AccAddressFromBech32(keepertest.Alice)
	reserveCoins := sdk.NewCoins(sdk.NewCoin("uwusdc", math.NewInt(10)))
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress()
	err := k.BankKeeper.SendCoins(ctx, addr, acc, reserveCoins)
	require.NoError(t, err)

	price1, _ := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")

	_, err = keepertest.Sell(ctx, msg, &dextypes.MsgSell{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "1_000_000000",
	})
	require.NoError(t, err)

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.True(t, price2.GT(price1))

	priceBase, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, "uwusdc")
	require.NoError(t, err)
	require.False(t, priceBase.IsNil())

	parity, _, err := k.DenomKeeper.CalculateParity(ctx, constants.KUSD)
	require.NoError(t, err)
	require.False(t, parity.IsNil())
	require.True(t, parity.LT(math.LegacyOneDec()))

	for i := 0; i < 10; i++ {
		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			return k.Burn(innerCtx)
		}))

		var price3 math.LegacyDec
		price3, err = k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
		require.NoError(t, err)
		require.True(t, price3.LT(price2))
	}

	require.True(t, liquidityBalanced(ctx, dexK))
}

func addReserveFundsToDex(ctx context.Context, acc swaptypes.AccountKeeper, dex swaptypes.DexKeeper, bank swaptypes.BankKeeper, t *testing.T, denom string, amount int64) {
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

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return dex.AddLiquidity(innerCtx, reserveAcc.GetAddress(), denom, math.LegacyNewDec(amount).RoundInt())
	}))
}

func TestBurn3(t *testing.T) {
	supply1 := burnScenario(t, 1_000_000000)
	supply2 := burnScenario(t, 10_000_000000)

	// not exactly equal because of rounding
	diff := gomath.Abs(float64(supply1 - supply2))
	require.True(t, diff < 5)
}

func burnScenario(t *testing.T, sellAmount int64) int64 {
	k, _, dexK, _, ctx := keepertest.SetupSwapMsgServer(t)

	addr, err := sdk.AccAddressFromBech32(keepertest.Alice)
	require.NoError(t, err)

	reserveAcc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress().String()

	bankKeeper := k.BankKeeper.(keepertest.TestBankKeeper)
	keepertest.AddFunds(ctx, t, bankKeeper, constants.BaseCurrency, reserveAcc, 100_000)
	keepertest.AddFunds(ctx, t, bankKeeper, constants.KUSD, reserveAcc, 100_000)
	keepertest.AddFunds(ctx, t, bankKeeper, "uwusdc", reserveAcc, 100_000)

	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, reserveAcc, constants.BaseCurrency, 100_000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, reserveAcc, constants.KUSD, 100_000)
	keepertest.TestAddLiquidity(ctx, k.DexKeeper, t, reserveAcc, "uwusdc", 100_000)
	addReserveFundsToDex(ctx, k.AccountKeeper, k.DexKeeper, k.BankKeeper, t, constants.KUSD, 10)

	tradeCtx := dextypes.TradeContext{
		Context:             ctx,
		CoinSource:          addr.String(),
		CoinTarget:          addr.String(),
		TradeAmount:         math.NewInt(sellAmount),
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: "uwusdc",
		MaxPrice:            nil,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	var tradeResult trading.TradeResult
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.DexKeeper.ExecuteSell(tradeCtx)
		return err
	}))

	require.True(t, tradeResult.AmountGiven().IsPositive())

	parity1, _, err := k.DenomKeeper.CalculateParity(ctx, constants.KUSD)
	require.NoError(t, err)
	require.True(t, parity1.LT(math.LegacyOneDec()))

	maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, constants.KUSD)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.CheckBurn(innerCtx, constants.KUSD, maxBurnAmount)
	}))
	_, err = k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.True(t, liquidityBalanced(ctx, dexK))

	return k.BankKeeper.GetSupply(ctx, constants.BaseCurrency).Amount.Int64()
}

func liquidityBalanced(ctx context.Context, k dexkeeper.Keeper) bool {
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liqSum := k.GetPoolLiquidity(ctx, denom)
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
