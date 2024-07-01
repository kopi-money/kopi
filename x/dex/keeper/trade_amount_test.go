package keeper_test

import (
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/dex/types"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
)

func TestTradeAmount1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)

	offer := math.NewInt(10_000)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	tradeCtx := types.TradeContext{
		CoinSource:       keepertest.Bob,
		CoinTarget:       keepertest.Bob,
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: keeper.FlatPrice{},
		AllowIncomplete:  false,
	}

	var amountReceived math.Int
	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, amountReceived, _, _, err = k.ExecuteTrade(tradeCtx)
		return err
	}))

	tradeAmount1 := k.GetTradeAmount(ctx, acc.String())
	require.Equal(t, tradeAmount1.Amount, amountReceived.ToLegacyDec())

	require.NoError(t, cache.Transact(ctx, func(innerCtx sdk.Context) error {
		tradeCtx.Context = innerCtx
		_, _, amountReceived, _, _, err = k.ExecuteTrade(tradeCtx)
		return err
	}))

	_ = cache.Transact(ctx, func(innerCtx sdk.Context) error {
		k.TradeAmountDecay(innerCtx)
		return nil
	})

	tradeAmount2 := k.GetTradeAmount(ctx, acc.String())
	require.True(t, tradeAmount2.Amount.LT(tradeAmount1.Amount))
}
