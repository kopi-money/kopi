package keeper_test

import (
	"context"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConversions1(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))

	require.ErrorIs(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
		})

		return err
	}), types.ErrTokenConversionListEmpty)

	require.ErrorIs(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            factoryDenomHash,
					ConversationRate: "1",
				},
			},
		})

		return err
	}), types.ErrTokenConversionSameToken)

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            constants.KUSD,
					ConversationRate: "0",
				},
			},
		})

		return err
	}), "invalid conversion rate")

	require.ErrorIs(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            constants.KUSD,
					ConversationRate: "1",
				},
				{
					Denom:            constants.KUSD,
					ConversationRate: "1",
				},
			},
		})

		return err
	}), types.ErrTokenConversionDuplicateToken)
}

func TestConversions2(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash1, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom1", "test1", 6)
	require.NoError(t, err)
	factoryDenomHash2, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom2", "test2", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash1, keepertest.Alice, "100"))
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash2, keepertest.Alice, "100"))

	require.ErrorIs(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash1,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            factoryDenomHash2,
					ConversationRate: "1",
				},
			},
		})

		return err
	}), types.ErrTokenConversionMintable)

	require.NoError(t, keepertest.DisableMinting(ctx, msgServer, keepertest.Alice, factoryDenomHash2))
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash1,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            factoryDenomHash2,
					ConversationRate: "1",
				},
			},
		})

		return err
	}))
}

func TestConversions3(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash1, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom1", "test1", 6)
	require.NoError(t, err)
	factoryDenomHash2, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom2", "test2", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash1, keepertest.Alice, "100"))
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash2, keepertest.Alice, "100"))

	require.NoError(t, keepertest.DisableMinting(ctx, msgServer, keepertest.Alice, factoryDenomHash2))
	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash1,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            factoryDenomHash2,
					ConversationRate: "2",
				},
			},
		})

		return err
	}), "not enough funds for")
}

func TestConversions4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            constants.KUSD,
					ConversationRate: "1",
				},
			},
		})

		return err
	}))

	conversion, has := k.GetTokenConversion(ctx, factoryDenomHash)
	require.True(t, has)
	require.Len(t, conversion.Entries, 1)
	require.Equal(t, conversion.Entries[0].NewToken, constants.KUSD)
	require.Equal(t, conversion.Entries[0].ConversionRate, math.LegacyOneDec())

	moduleAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolConversions)
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), constants.KUSD).Amount.Int64(), int64(100))
}

func TestConversions5(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            constants.KUSD,
					ConversationRate: "1",
				},
			},
		})

		return err
	}))

	require.Error(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "200"))
	require.NoError(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100"))

	moduleAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolConversions)
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), constants.KUSD).Amount.Int64(), int64(0))
}

func TestConversions6(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            constants.KUSD,
					ConversationRate: "1",
				},
			},
		})

		return err
	}))

	require.NoError(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "50"))

	moduleAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolConversions)
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), constants.KUSD).Amount.Int64(), int64(50))

	require.Error(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "100"))

	require.NoError(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "50"))
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), constants.KUSD).Amount.Int64(), int64(0))
}

func TestConversions7(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	factoryDenomHash1, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom1", "test1", 6)
	require.NoError(t, err)
	factoryDenomHash2, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom2", "test2", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "100"))
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash1, keepertest.Alice, "1000"))
	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash2, keepertest.Alice, "10"))

	require.NoError(t, keepertest.DisableMinting(ctx, msgServer, keepertest.Alice, factoryDenomHash1))
	require.NoError(t, keepertest.DisableMinting(ctx, msgServer, keepertest.Alice, factoryDenomHash2))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateTokenConversion(innerCtx, &types.MsgCreateTokenConversion{
			Creator:              keepertest.Alice,
			FullFactoryDenomName: factoryDenomHash,
			Conversions: []types.TokenConversionCreation{
				{
					Denom:            factoryDenomHash1,
					ConversationRate: "10",
				},
				{
					Denom:            factoryDenomHash2,
					ConversationRate: "0.1",
				},
			},
		})

		return err
	}))

	moduleAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolConversions)
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash1).Amount.Int64(), int64(1000))
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash2).Amount.Int64(), int64(10))

	require.NoError(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "10"))

	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash1).Amount.Int64(), int64(900))
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash2).Amount.Int64(), int64(9))

	require.NoError(t, keepertest.ConvertTokens(ctx, msgServer, keepertest.Alice, factoryDenomHash, "90"))

	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash1).Amount.Int64(), int64(0))
	require.Equal(t, k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), factoryDenomHash2).Amount.Int64(), int64(0))
}
