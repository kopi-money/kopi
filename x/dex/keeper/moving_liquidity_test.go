package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestMovingLiquidity1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 50047614030762)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 5326805883)
	keepertest.AddFunds(ctx, t, k.BankKeeper, "uwusdc", keepertest.Alice, 77569727)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 50_000000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 70_000000))
	setMovingLiqFixed(ctx, k)

	amount := math.NewInt(50_000000)
	zeroDec := math.LegacyZeroDec()

	tradeContext := types.TradeContext{
		TradeAmount:         amount,
		Fee:                 &zeroDec,
		Context:             ctx,
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.UpdateMovingLiquidities(innerCtx)
		return nil
	})

	res1, err := k.SimulateSell(ctx, amount, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	pricePaid1, err := res1.PricePaid()
	require.NoError(t, err)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 70_000000))

	res2, err := k.SimulateSell(tradeContext, amount, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	pricePaid2, err := res2.PricePaid()
	require.NoError(t, err)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.UpdateMovingLiquidities(innerCtx)
		return nil
	})

	res3, err := k.SimulateSell(tradeContext, amount, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	pricePaid3, err := res3.PricePaid()
	require.NoError(t, err)

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.UpdateMovingLiquidities(innerCtx)
		return nil
	})

	res4, err := k.SimulateSell(tradeContext, amount, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	pricePaid4, err := res4.PricePaid()
	require.NoError(t, err)

	require.NoError(t, keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 70_000000))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.UpdateMovingLiquidities(innerCtx)
		return nil
	})

	res5, err := k.SimulateSell(tradeContext, amount, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	pricePaid5, err := res5.PricePaid()
	require.NoError(t, err)

	fmt.Println(pricePaid1)
	fmt.Println(pricePaid2)
	fmt.Println(pricePaid3)
	fmt.Println(pricePaid4)
	fmt.Println(pricePaid5)

	require.True(t, pricePaid1.Equal(pricePaid2))
	require.True(t, pricePaid1.GT(pricePaid3))
	require.True(t, pricePaid3.GT(pricePaid4))
	require.True(t, pricePaid1.Equal(pricePaid5))
}
