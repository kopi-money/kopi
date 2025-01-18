package keeper_test

import (
	"context"
	"testing"

	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
)

func TestTradeAmount1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))

	maximumTradable, _ := k.CalculateMaximumSellableAmount(types.TradeContext{
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		OrdersCaches:        k.NewOrdersCaches(ctx),
	})
	require.Nil(t, maximumTradable)
}

func TestTradeAmount2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 500_000))

	offer := math.NewInt(10_000)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	tradeCtx := types.TradeContext{
		CoinSource:          keepertest.Bob,
		CoinTarget:          keepertest.Bob,
		TradeAmount:         offer,
		TradeDenomGiving:    constants.KUSD,
		TradeDenomReceiving: constants.BaseCurrency,
		MinimumTradeAmount:  &offer,
		TradeBalances:       keeper.NewTradeBalances(),
		Fee:                 math.LegacyZeroDec(),
	}

	var (
		tradeResult types.TradeResult
		err         error
	)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	tradeAmount1 := k.GetTradeAmount(ctx, acc.String())
	require.Equal(t, tradeAmount1, tradeResult.AmountReceived.ToLegacyDec())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		tradeResult, err = k.ExecuteSell(tradeCtx)
		return err
	}))

	_ = cache.Transact(ctx, func(innerCtx context.Context) error {
		k.TradeAmountDecay(innerCtx)
		return nil
	})

	tradeAmount2 := k.GetTradeAmount(ctx, acc.String())
	require.True(t, tradeAmount2.LT(tradeAmount1))
}
