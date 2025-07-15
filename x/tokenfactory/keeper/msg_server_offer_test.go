package keeper_test

import (
	"context"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOffers1(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateOffers(innerCtx, &types.MsgCreateOffers{
			Creator:              keepertest.Alice,
			Receivers:            []string{},
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   "1000",
			AskAmount:            "ukusd",
			AskDenom:             "1000",
			NumUnlockSteps:       1,
			OpenOffer:            false,
		})

		return err
	}))
}

func TestOffers2(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateOffers(innerCtx, &types.MsgCreateOffers{
			Creator:              keepertest.Alice,
			Receivers:            []string{keepertest.Bob},
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   "1000",
			AskAmount:            "ukusd",
			AskDenom:             "1000",
			NumUnlockSteps:       1,
			OpenOffer:            true,
		})

		return err
	}))
}

func TestOffers3(t *testing.T) {
	_, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateOffers(innerCtx, &types.MsgCreateOffers{
			Creator:              keepertest.Alice,
			Receivers:            []string{},
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   "1000",
			AskAmount:            "1000",
			AskDenom:             constants.KUSD,
			NumUnlockSteps:       1,
			OpenOffer:            true,
		})

		return err
	}))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOffer(innerCtx, &types.MsgTakeOffer{
			Creator:    keepertest.Bob,
			OfferIndex: 1,
		})

		return err
	}))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOpenOffer(innerCtx, &types.MsgTakeOpenOffer{
			Creator:    keepertest.Bob,
			Amount:     "0",
			OfferIndex: 1,
		})

		return err
	}))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOpenOffer(innerCtx, &types.MsgTakeOpenOffer{
			Creator:    keepertest.Bob,
			Amount:     "2000",
			OfferIndex: 1,
		})

		return err
	}))
}

func TestOffers4(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateOffers(innerCtx, &types.MsgCreateOffers{
			Creator:              keepertest.Alice,
			Receivers:            []string{},
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   "1000",
			AskAmount:            "1000",
			AskDenom:             constants.KUSD,
			NumUnlockSteps:       1,
			OpenOffer:            true,
		})

		return err
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Bob, 500)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOpenOffer(innerCtx, &types.MsgTakeOpenOffer{
			Creator:    keepertest.Bob,
			Amount:     "500",
			OfferIndex: 1,
		})

		return err
	}))

	offer, has := k.GetOffer(ctx, 1)
	require.True(t, has)

	require.Equal(t, int64(500), offer.AskAmount.Int64())
	require.Equal(t, int64(500), offer.FactoryDenomAmount.Int64())

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOpenOffer(innerCtx, &types.MsgTakeOpenOffer{
			Creator:    keepertest.Bob,
			Amount:     "500",
			OfferIndex: 1,
		})

		return err
	}))

	_, has = k.GetOffer(ctx, 1)
	require.False(t, has)
}

func TestOffers5(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	factoryDenomHash, err := keepertest.CreateFactoryDenom(ctx, msgServer, keepertest.Alice, "testdenom", "test", 6)
	require.NoError(t, err)

	require.NoError(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "2000"))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.CreateOffers(innerCtx, &types.MsgCreateOffers{
			Creator:              keepertest.Alice,
			Receivers:            []string{},
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   "2000",
			AskAmount:            "1000",
			AskDenom:             constants.KUSD,
			NumUnlockSteps:       1,
			OpenOffer:            true,
		})

		return err
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Bob, 500)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.TakeOpenOffer(innerCtx, &types.MsgTakeOpenOffer{
			Creator:    keepertest.Bob,
			Amount:     "500",
			OfferIndex: 1,
		})

		return err
	}))

	offer, has := k.GetOffer(ctx, 1)
	require.True(t, has)

	require.Equal(t, int64(500), offer.AskAmount.Int64())
	require.Equal(t, int64(1000), offer.FactoryDenomAmount.Int64())
}
