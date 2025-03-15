package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/stretchr/testify/require"
)

func TestRatios1(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	denomMsg := denomkeeper.NewMsgServerImpl(k)

	// Add BTC with a price 1 BTC = 1000 kUSD
	require.NoError(t, keepertest.AddDexDenom(ctx, denomMsg, &denomtypes.MsgDexAddDenom{
		Authority:    k.GetAuthority(),
		Name:         "bitcoin",
		Factor:       "1000ukusd",
		MinLiquidity: "1000000",
		MinOrderSize: "1000000",
		Exponent:     8,
	}))

	ratio, err := k.GetRatio(ctx, "bitcoin")
	require.NoError(t, err)

	// 1 BTC = 1000 kUSD = 4000 XKP
	// 1 / 4000 = 0.00025
	require.Equal(t, math.LegacyNewDecWithPrec(25, 3), ratio.Ratio)
}

func TestRatios2(t *testing.T) {
	k, ctx, _ := keepertest.DenomKeeper(t)
	denomMsg := denomkeeper.NewMsgServerImpl(k)

	// Add BTC with a price 1 BTC = 1000 kUSD
	require.NoError(t, keepertest.AddDexDenom(ctx, denomMsg, &denomtypes.MsgDexAddDenom{
		Authority:    k.GetAuthority(),
		Name:         "inj2",
		Factor:       "22ukusd",
		MinLiquidity: "1000000",
		MinOrderSize: "1000000",
		Exponent:     18,
	}))
}
