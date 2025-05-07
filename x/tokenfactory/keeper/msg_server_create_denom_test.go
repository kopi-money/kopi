package keeper_test

import (
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/keeper"
	"github.com/stretchr/testify/require"
)

func TestCreateDenom1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	_, has := k.GetDenom(ctx, keepertest.Alice, "testdenom")
	require.False(t, has)

	_, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	_, has = k.GetDenom(ctx, keepertest.Alice, "testdenom")
	require.False(t, has)
	_, has = k.GetDenom(ctx, keepertest.Alice, "test")
	require.True(t, has)

	_, has = k.GetDenom(ctx, keepertest.Alice, "test2")
	require.False(t, has)
	_, has = k.GetDenomByFullName(ctx, keeper.ToFullName(keepertest.Alice, "test2"))
	require.False(t, has)
}

func TestCreateDenom2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	_, has := k.GetDenom(ctx, keepertest.Alice, "testdenom")
	require.False(t, has)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	_, has = k.GetDenom(ctx, keepertest.Alice, "testdenom")
	require.False(t, has)
	_, has = k.GetDenom(ctx, keepertest.Alice, "test")
	require.True(t, has)
	_, has = k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	_, has = k.GetDenom(ctx, keepertest.Alice, "testdenom2")
	require.False(t, has)
	_, has = k.GetDenomByFullName(ctx, keeper.ToFullName(keepertest.Alice, "testdenom2"))
	require.False(t, has)
}
