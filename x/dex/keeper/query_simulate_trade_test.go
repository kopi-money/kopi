package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestSimulateTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000_000))

	_, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Alice,
		Amount:         "9999999999",
	})

	require.NoError(t, err)

	_, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Alice,
		Amount:         "10000000000",
	})

	require.Error(t, err)

	_, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Alice,
		Amount:         "10000000001",
	})

	require.Error(t, err)
}
