package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestSimulateTrade1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000_000000))
	setMovingLiqFixed(ctx, k)

	_, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Bob,
		Amount:         "2_500_000000",
	})
	require.Error(t, err)

	_, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Bob,
		Amount:         "2_499_999999",
	})
	require.NoError(t, err)

	_, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Bob,
		Amount:         "10_000_000000",
	})
	require.Error(t, err)

	_, err = k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Bob,
		Amount:         "10000000001",
	})
	require.Error(t, err)
}

func TestSimulateTrade2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("10000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_463686_945231)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 64_471_465592)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_463686_945231))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 64_471_465592))
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msg.AddLiquidity(innerCtx, &types.MsgAddLiquidity{
			Creator: keepertest.Alice,
			Denom:   "inj",
			Amount:  "10_000000_000000_000000",
		})
		return err
	}))

	setMovingLiqFixed(ctx, k)

	buyAmount := "2000000000000000000"
	res, err := k.QuerySimulateBuy(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    constants.KUSD,
		DenomReceiving: "inj",
		Address:        keepertest.Alice,
		Amount:         buyAmount,
	})

	require.NoError(t, err)
	require.Equal(t, buyAmount, res.AmountReceived)
}

func TestSimulateTrade3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	amount, ok := math.NewIntFromString("10000000000000000000")
	require.True(t, ok)
	keepertest.AddFundsInt(ctx, t, k.BankKeeper, "inj", keepertest.Alice, amount)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Alice, 4_463686_945231)
	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Alice, 64_471_465592)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 4_463686_945231))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 64_471_465592))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msg.AddLiquidity(innerCtx, &types.MsgAddLiquidity{
			Creator: keepertest.Alice,
			Denom:   "inj",
			Amount:  "10_000000_000000_000000",
		})
		return err
	}))

	sellAmount := "2000000000000000000"
	res, err := k.QuerySimulateSell(ctx, &types.QuerySimulateTradeRequest{
		DenomGiving:    "inj",
		DenomReceiving: constants.KUSD,
		Address:        keepertest.Alice,
		Amount:         sellAmount,
	})

	require.NoError(t, err)
	require.Equal(t, sellAmount, res.AmountGiven)
}
