package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestMint1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))
	minted := k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	require.Equal(t, int64(100), minted.Int64())

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))
	minted = k.BankKeeper.GetSupply(ctx, factoryDenomHash).Amount
	require.Equal(t, int64(200), minted.Int64())
}

func TestMint2(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Bob, "100"))

	acc, _ := sdk.AccAddressFromBech32(keepertest.Bob)
	balance := k.BankKeeper.SpendableCoin(ctx, acc, factoryDenomHash).Amount
	require.Equal(t, int64(100), balance.Int64())
}

func TestMint3(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.Error(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Bob, factoryDenomHash, keepertest.Bob, "100"))
}
