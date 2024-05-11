package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestTradeAmount1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 500_000)
	require.NoError(t, err)

	offer := math.NewInt(10_000)

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	options := types.TradeOptions{
		CoinSource:       acc,
		CoinTarget:       acc,
		GivenAmount:      offer,
		TradeDenomStart:  "ukusd",
		TradeDenomEnd:    utils.BaseCurrency,
		TradeCalculation: keeper.FlatPrice{},
		AllowIncomplete:  false,
		LiquidityMap:     make(types.LiquidityMap),
	}

	_, amountReceived, _, _, err := k.ExecuteTrade(ctx, ctx.EventManager(), options)

	tradeAmount1 := k.GetTradeAmount(ctx, acc.String())
	require.Equal(t, tradeAmount1.Amount, amountReceived.ToLegacyDec())

	_, amountReceived, _, _, err = k.ExecuteTrade(ctx, ctx.EventManager(), options)

	k.TradeAmountDecay(ctx)

	tradeAmount2 := k.GetTradeAmount(ctx, acc.String())
	require.True(t, tradeAmount2.Amount.LT(tradeAmount1.Amount))

	require.True(t, liquidityMapCorrect(ctx, k, options.LiquidityMap))
}
