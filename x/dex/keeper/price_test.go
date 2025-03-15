package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestPrice1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000)
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000)
	require.Nil(t, err)

	price1, err := k.DenomKeeper.CalculatePrice(ctx, constants.BaseCurrency, constants.KUSD)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(4), price1)

	price2, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, constants.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), price2)

	price3, err := k.DenomKeeper.CalculatePrice(ctx, constants.KUSD, "uwusdc")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(1), price3)
}
