package keeper_test

import (
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestBurn1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))

	require.NoError(t, keepertest.BurnFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, "50"))
	supply := k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	require.Equal(t, int64(50), supply.Int64())

	require.NoError(t, keepertest.BurnFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, "50"))
	supply = k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	require.Equal(t, int64(0), supply.Int64())
}

func TestBurn2(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenom, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenom, keepertest.Alice, "100"))
}

func TestBurn3(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenom, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenom, keepertest.Alice, "100"))
	require.Error(t, keepertest.BurnFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenom, "200"))
}

func TestBurn4(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenom, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenom, keepertest.Bob, "100"))
	require.NoError(t, keepertest.BurnFactoryDenom(ctx, msgServer, keepertest.Bob, factoryDenom, "100"))
}
