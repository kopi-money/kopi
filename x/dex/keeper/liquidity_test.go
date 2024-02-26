package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLiquidity1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	require.Nil(t, err)
	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(1), liq)

	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	require.Nil(t, err)
	num := k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 2, num)
	liq, found = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(2), liq)
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, math.NewInt(2), liq)

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 1, num)
	liq, found = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(1), liq)
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, math.NewInt(1), liq)

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	num = k.GetLiquidityEntriesByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, 0, num)
	liq, found = k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, math.NewInt(0), liq)
	liq = k.GetLiquidityByAddress(ctx, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, math.NewInt(0), liq)
}

func TestLiquidity2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Bob, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Carol, utils.BaseCurrency, 1)

	var idx uint64 = 0
	iterator := k.LiquidityIterator(ctx, utils.BaseCurrency)
	for ; iterator.Valid(); iterator.Next() {
		liq := k.LiquidityUnmarshal(iterator.Value())

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}

	_ = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)
	_ = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1)

	idx = 0
	iterator = k.LiquidityIterator(ctx, utils.BaseCurrency)
	for ; iterator.Valid(); iterator.Next() {
		liq := k.LiquidityUnmarshal(iterator.Value())

		require.Less(t, idx, liq.Index)
		idx = liq.Index
	}
}

func TestLiquidity3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount1 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)

	liq, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	require.True(t, found)
	require.Equal(t, liq, math.NewInt(10))

	amount2 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount2.Add(math.NewInt(10)))
	require.True(t, amount2.LT(amount1))

	err = keepertest.RemoveLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 10)
	require.NoError(t, err)

	amount3 := getSpendableAmount(ctx, k, utils.BaseCurrency, keepertest.Alice)
	require.Equal(t, amount1, amount3)
}

func getSpendableAmount(ctx sdk.Context, k keeper.Keeper, denom, address string) math.Int {
	addr, _ := sdk.AccAddressFromBech32(address)
	coins := k.BankKeeper.SpendableCoins(ctx, addr)

	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.Int{}
}
