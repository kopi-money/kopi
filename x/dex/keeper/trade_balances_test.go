package keeper_test

import (
	"cosmossdk.io/math"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTradeBalances1(t *testing.T) {
	acc1 := keepertest.Alice
	acc2 := keepertest.Bob

	tb := dexkeeper.NewTradeBalances()
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(100))

	transfers, err := tb.MergeTransfers()
	require.NoError(t, err)
	require.Equal(t, 1, len(transfers))
	require.False(t, transfers[0].Amount.IsNil())
	require.Equal(t, int64(100), transfers[0].Amount.Int64())
}

func TestTradeBalances2(t *testing.T) {
	acc1 := keepertest.Alice
	acc2 := keepertest.Bob

	tb := dexkeeper.NewTradeBalances()
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(100))
	tb.AddTransfer(acc2, acc1, "ukusd", math.NewInt(100))

	transfers, err := tb.MergeTransfers()
	require.NoError(t, err)
	require.Equal(t, 0, len(transfers))
}

func TestTradeBalances3(t *testing.T) {
	acc1 := keepertest.Alice
	acc2 := keepertest.Bob

	tb := dexkeeper.NewTradeBalances()
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(100))
	tb.AddTransfer(acc2, acc1, "ukusd", math.NewInt(50))

	transfers, err := tb.MergeTransfers()
	require.NoError(t, err)
	require.Equal(t, 1, len(transfers))
	require.Equal(t, int64(50), transfers[0].Amount.Int64())
}

func TestTradeBalances4(t *testing.T) {
	acc1 := keepertest.Alice
	acc2 := keepertest.Bob

	tb := dexkeeper.NewTradeBalances()
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(100))
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(50))

	transfers, err := tb.MergeTransfers()
	require.NoError(t, err)
	require.Equal(t, 1, len(transfers))
	require.Equal(t, int64(150), transfers[0].Amount.Int64())
}

func TestTradeBalances5(t *testing.T) {
	acc1 := keepertest.Alice
	acc2 := keepertest.Bob
	acc3 := keepertest.Carol

	tb := dexkeeper.NewTradeBalances()
	tb.AddTransfer(acc1, acc2, "ukusd", math.NewInt(100))
	tb.AddTransfer(acc2, acc3, "ukusd", math.NewInt(100))

	transfers, err := tb.MergeTransfers()
	require.NoError(t, err)
	require.Equal(t, 1, len(transfers))
	require.Equal(t, int64(100), transfers[0].Amount.Int64())
	require.Equal(t, acc1, transfers[0].From)
	require.Equal(t, acc3, transfers[0].To)
}
