package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/stretchr/testify/require"
)

func TestPrice1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.Nil(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.Nil(t, err)

	price1, err := k.CalculatePrice(ctx, utils.BaseCurrency, "ukusd")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(4), price1)

	price2, err := k.CalculatePrice(ctx, "ukusd", utils.BaseCurrency)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), price2)

	price3, err := k.CalculatePrice(ctx, "ukusd", "uwusdc")
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(1), price3)
}
