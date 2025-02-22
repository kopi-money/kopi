package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestTradeSimulation1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "4000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "4000000", constants.KUSD, "1000000", "0.1", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(4_000_000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(1_000_000), pool.KCoinAmount.Int64())

	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Address:        keepertest.Alice,
		Amount:         "1_000",
	})

	require.NoError(t, err)
	require.Equal(t, "1000", res.AmountGiven)
	require.Equal(t, "3592", res.AmountReceived)

	res, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: factoryDenomHash,
		Address:        keepertest.Alice,
		Amount:         "1_000",
	})

	require.NoError(t, err)
	require.Equal(t, "1000", res.AmountReceived)
	require.Equal(t, "275", res.AmountGiven)
}
