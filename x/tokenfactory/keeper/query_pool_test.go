package keeper_test

import (
	"testing"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestPoolLiquidityAddress1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "4000000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "4000000", constants.KUSD, "1000000", "0.01", 10))

	res, err := k.QueryPoolLiquidityAddress(ctx, &types.QueryPoolLiquidityAddressRequest{
		Address: keepertest.Alice,
	})

	require.NoError(t, err)
	require.Len(t, res.Pools, 1)

	require.Equal(t, res.Pools[0].FactoryDenomHash, factoryDenomHash)
	require.Equal(t, res.Pools[0].AmountKcoin, "1000000")
	require.Equal(t, res.Pools[0].AmountFactoryToken, "4000000")
	require.Equal(t, res.Pools[0].LiquidityValue, "2000000.000000000000000000")
}
