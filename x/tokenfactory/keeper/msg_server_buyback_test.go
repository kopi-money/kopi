package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/constants"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestBuyback1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000000", constants.KUSD, "1000000", "0.01", 10))

	pool, _ := k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1000000), pool.FactoryDenomAmount.Int64())
	require.Equal(t, int64(1000000), pool.KCoinAmount.Int64())

	supply1 := k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	res, err := keepertest.FactoryDenomBuyback(ctx, msgServer, keepertest.Alice, factoryDenomHash, "10000")
	require.NoError(t, err)

	require.Equal(t, "10000", res.BuybackAmount)
	require.Equal(t, "9802", res.BurnedAmount)

	pool, _ = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.Equal(t, int64(1009950), pool.KCoinAmount.Int64())
	require.Equal(t, int64(990198), pool.FactoryDenomAmount.Int64())

	supply2 := k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	require.Less(t, supply2.Int64(), supply1.Int64())
}
