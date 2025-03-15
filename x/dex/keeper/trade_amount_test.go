package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/x/dex/types"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
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
