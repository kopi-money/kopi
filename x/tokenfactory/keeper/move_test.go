package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"

	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestMove1(t *testing.T) {
	k, msgServer, ctx := keepertest.SetupTokenfactoryMsgServer(t)

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		acc, _ := sdk.AccAddressFromBech32(keepertest.Alice)
		return k.DexKeeper.AddLiquidityWithCompound(innerCtx, acc, constants.BaseCurrency, math.NewInt(100), true)
	}))

	localName := "ibc/1E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5"
	keepertest.AddFunds(ctx, t, k.BankKeeper, localName, keepertest.Alice, 1000)

	factoryDenomHash, err := keepertest.CreateFactoryDenomFromLocal(ctx, msgServer, keepertest.Alice, "testdenom", localName, "test", 2, 6)
	require.NoError(t, err)

	require.Error(t, keepertest.MintFactoryDenom(ctx, msgServer, keepertest.Alice, factoryDenomHash, keepertest.Alice, "1000"))
	require.NoError(t, keepertest.CreatePool(ctx, msgServer, keepertest.Alice, factoryDenomHash, "1000", constants.KUSD, "1000", "0.01", 10))

	factoryDenom, has := k.GetDenomByFullName(ctx, factoryDenomHash)
	require.True(t, has)

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.MoveDenom(innerCtx, factoryDenom)
	}))

	keepertest.AddFunds(ctx, t, k.BankKeeper, localName, keepertest.Alice, 5000_000000)
	require.NoError(t, keepertest.AddOneSidedFactoryLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "5000_000000"))
	require.NoError(t, keepertest.AddOneSidedKCoinLiquidity(ctx, msgServer, keepertest.Alice, factoryDenomHash, "5000_000000"))

	require.ErrorContains(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.MoveDenom(innerCtx, factoryDenom)
	}), "pool value too small")

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		now := time.Now()
		pool, _ := k.GetLiquidityPool(innerCtx, factoryDenomHash)
		pool.ThresholdCrossed = &now
		k.SetLiquidityPool(innerCtx, factoryDenomHash, pool)

		return k.MoveDenom(innerCtx, factoryDenom)
	}))

	require.Error(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.MoveDenom(innerCtx, factoryDenom)
	}))

	_, has = k.GetLiquidityPool(ctx, factoryDenomHash)
	require.False(t, has)

	denomKeeper := k.DenomKeeper.(denomkeeper.Keeper)
	ratio, err := denomKeeper.GetRatio(ctx, localName)
	require.NoError(t, err)
	require.Equal(t, "0.250000000000000000", ratio.Ratio.String())

	dexKeeper := k.DexKeeper.(dexkeeper.Keeper)
	localAmount := dexKeeper.GetPoolLiquidity(ctx, localName)
	require.Equal(t, int64(5000_001000), localAmount.Int64())
}
