package keeper_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestOrders1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	require.Error(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
	}))

	require.Error(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "abc",
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "1",
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	poolBalance := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())
	require.Equal(t, int64(2_000_000), poolBalance.AmountOf(constants.BaseCurrency).Int64())
	require.Equal(t, int64(2_000_000), poolBalance.AmountOf(constants.KUSD).Int64())

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "0.5",
		Blocks:         1000,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 0, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func executeOrders(ctx context.Context, k dexkeeper.Keeper) error {
	return cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		eventManager := sdk.UnwrapSDKContext(innerCtx).EventManager()
		blockHeight := sdk.UnwrapSDKContext(innerCtx).BlockHeight()

		return k.ExecuteOrders(innerCtx, eventManager, blockHeight)
	})
}

func executeOrder(ctx context.Context, k dexkeeper.Keeper, order *types.Order) (types.TradeResult, bool, error) {
	var (
		tradeResult   types.TradeResult
		fullyExecuted bool
	)

	err := cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		fee := k.GetJoinedFee(ctx)
		var innerErr error

		tradeResult, fullyExecuted, innerErr = k.ExecuteOrder(innerCtx, k.NewOrdersCaches(ctx), fee, order)
		return innerErr
	})

	return tradeResult, fullyExecuted, err
}

func TestOrders3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "0.1",
		Blocks:         1000,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 0, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "0.1",
		Blocks:         10,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 1, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1",
		MaxPrice:       "0.1001",
		Blocks:         1000,
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100000",
		MaxPrice:       "0.1",
		Blocks:         1000,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 1, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "0.1",
		Blocks:         1000,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 0, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1",
		MaxPrice:       "10",
		Blocks:         10,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 1, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1",
		MaxPrice:       "10",
	}))

	counter := 0

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()
		if order.Creator == keepertest.Bob {
			counter += 1
		}
	}
	require.Equal(t, 1, counter)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders10(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000)
	require.NoError(t, err)

	address, err := sdk.AccAddressFromBech32(keepertest.Bob)
	require.NoError(t, err)

	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.BaseCurrency)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1",
		MaxPrice:       "10",
		Blocks:         0,
	}))

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.BaseCurrency)
	kopi2 = kopi2.Add(math.NewInt(1))

	require.Equal(t, kopi1, kopi2)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000)
	require.NoError(t, err)

	address, err := sdk.AccAddressFromBech32(keepertest.Bob)
	require.NoError(t, err)

	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.BaseCurrency)
	kusd1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.KUSD)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "1000",
		MaxPrice:       "0.0001",
		Blocks:         100,
	}))

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.BaseCurrency)
	require.True(t, kopi2.LT(kopi1))

	require.NoError(t, executeOrders(ctx, k))

	kusd2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), constants.KUSD)
	require.True(t, kusd2.GT(kusd1))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func getCoins(coins sdk.Coins, denom string) math.Int {
	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.ZeroInt()
}

func TestOrders12(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100",
		MaxPrice:       "1000",
		Blocks:         100,
	}))

	iterator := k.OrderIterator(ctx)
	orders := iterator.GetAll()
	require.Equal(t, 1, len(orders))
	order := orders[0]

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)
	require.True(t, kopi2.GT(kopi1))
	require.Equal(t, kopi2, math.NewInt(100))

	require.NoError(t, keepertest.RemoveOrder(ctx, msg, &types.MsgRemoveOrder{
		Creator: keepertest.Bob,
		Index:   order.Index,
	}))

	kopi3 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)
	require.Equal(t, kopi3, math.NewInt(0))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 2_000000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 2_000000))

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100",
		MaxPrice:       "1000",
		Blocks:         100,
	}))

	iterator := k.OrderIterator(ctx)
	orders := iterator.GetAll()
	require.Equal(t, 1, len(orders))
	order := orders[0]

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)
	require.True(t, kopi2.GT(kopi1))
	require.Equal(t, kopi2, math.NewInt(100))

	require.NoError(t, keepertest.RemoveOrder(ctx, msg, &types.MsgRemoveOrder{
		Creator: keepertest.Bob,
		Index:   order.Index,
	}))

	kopi3 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), constants.BaseCurrency)
	require.Equal(t, kopi3, math.NewInt(0))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000)
	require.NoError(t, err)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100_000",
		MaxPrice:       "1000",
		Blocks:         100,
	}))

	require.NoError(t, executeOrders(ctx, k))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000)
	require.NoError(t, err)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "100_000",
		MaxPrice:       "1000",
		Blocks:         100,
	}))

	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		eventManager := sdk.UnwrapSDKContext(innerCtx).EventManager()
		blockHeight := sdk.UnwrapSDKContext(innerCtx).BlockHeight()

		return k.ExecuteOrders(innerCtx, eventManager, blockHeight)
	}))

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000)
	require.NoError(t, err)

	for i := 0; i < 1000; i++ {
		am1 := randomAmount(10000)
		am2 := randomAmount(10000)

		require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomGiving:     constants.BaseCurrency,
			DenomReceiving:  constants.KUSD,
			Amount:          strconv.Itoa(am1),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		}))

		checkOrderPoolBalanceDiff(t, k, ctx)

		require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomGiving:     constants.KUSD,
			DenomReceiving:  constants.BaseCurrency,
			Amount:          strconv.Itoa(am2),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		}))

		checkOrderPoolBalanceDiff(t, k, ctx)

		require.NoError(t, executeOrders(ctx, k))
		checkOrderPoolBalanceDiff(t, k, ctx)

		require.True(t, liquidityBalanced(ctx, k))
		require.NoError(t, checkCache(ctx, k))
	}
}

func TestOrders17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 100))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100))

	for i := 0; i < 1000; i++ {
		require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomGiving:     "uwusdc",
			DenomReceiving:  constants.KUSD,
			Amount:          strconv.Itoa(randomAmount(1000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		}))

		require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomGiving:     constants.KUSD,
			DenomReceiving:  "uwusdc",
			Amount:          strconv.Itoa(randomAmount(1000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		}))

		require.NoError(t, executeOrders(ctx, k))

		require.True(t, liquidityBalanced(ctx, k))
		checkOrderPoolBalanceDiff(t, k, ctx)
		require.NoError(t, checkCache(ctx, k))
	}
}

func TestOrders18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     constants.BaseCurrency,
		DenomReceiving:  constants.KUSD,
		Amount:          "10000",
		TradeAmount:     "5000",
		MaxPrice:        "1000",
		Blocks:          100,
		AllowIncomplete: true,
	}))

	require.NoError(t, executeOrders(ctx, k))

	require.True(t, liquidityBalanced(ctx, k))
	checkOrderPoolBalanceDiff(t, k, ctx)
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders19(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1000))
	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     constants.KUSD,
		DenomReceiving:  constants.BaseCurrency,
		Amount:          "100000000",
		MaxPrice:        "1",
		Blocks:          100,
		AllowIncomplete: true,
	}))

	require.NoError(t, executeOrders(ctx, k))
	checkOrderPoolBalanceDiff(t, k, ctx)
	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders20(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.Error(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5",
		DenomReceiving:  constants.BaseCurrency,
		Amount:          "1",
		MaxPrice:        "1",
		Blocks:          100,
		AllowIncomplete: true,
	}))
}

func TestOrders21(t *testing.T) {
	_, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5",
		DenomReceiving:  constants.BaseCurrency,
		Amount:          "1000000",
		MaxPrice:        "1",
		Blocks:          100,
		AllowIncomplete: true,
	}))

	require.Error(t, keepertest.UpdateOrder(ctx, msg, &types.MsgUpdateOrder{
		Creator:  keepertest.Bob,
		Index:    1,
		Amount:   "1",
		MaxPrice: "1",
	}))
}

func TestOrders22(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	liqBase := k.LiquidityIterator(ctx, constants.BaseCurrency).GetAll()
	require.Equal(t, 1, len(liqBase))
	require.Equal(t, int64(10_000), liqBase[0].Amount.Int64())

	liqOther := k.LiquidityIterator(ctx, constants.KUSD).GetAll()
	require.Equal(t, 1, len(liqOther))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     constants.BaseCurrency,
		DenomReceiving:  constants.KUSD,
		Amount:          "1000",
		TradeAmount:     "1000",
		MaxPrice:        "0.0001",
		Blocks:          10000,
		Interval:        1,
		AllowIncomplete: true,
	}))

	require.NoError(t, executeOrders(ctx, k))

	liqBase = k.LiquidityIterator(ctx, constants.BaseCurrency).GetAll()
	require.Equal(t, 2, len(liqBase))
	require.Equal(t, int64(10_000), liqBase[0].Amount.Int64())
	require.Equal(t, int64(995), liqBase[1].Amount.Int64())

	liqOther = k.LiquidityIterator(ctx, constants.KUSD).GetAll()
	require.Equal(t, 1, len(liqOther))
	require.Equal(t, int64(9_759), liqOther[0].Amount.Int64())

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders23(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     constants.BaseCurrency,
		DenomReceiving:  constants.KUSD,
		Amount:          strconv.Itoa(1900),
		MaxPrice:        "1000",
		Blocks:          100,
		AllowIncomplete: true,
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomGiving:     constants.KUSD,
		DenomReceiving:  constants.BaseCurrency,
		Amount:          strconv.Itoa(7298),
		MaxPrice:        "1000",
		Blocks:          100,
		AllowIncomplete: true,
	}))

	require.NoError(t, executeOrders(ctx, k))
	checkOrderPoolBalanceDiff(t, k, ctx)

	require.True(t, liquidityBalanced(ctx, k))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders24(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Bob,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "1",
		Blocks:         1000,
	}))

	require.NoError(t, executeOrders(ctx, k))

	numOrders := len(k.OrderIterator(ctx).GetAll())
	require.Equal(t, 0, numOrders)

	require.True(t, liquidityBalanced(ctx, k))
	require.True(t, checkOrderPoolBalanced(k, ctx))
	require.NoError(t, checkCache(ctx, k))
}

func TestOrders25(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "ukusd", keepertest.Dave, 10_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "1000",
		MaxPrice:       "4",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)

	require.Equal(t, order.AmountLocked.Int64(), int64(4025))
	require.NoError(t, executeOrders(ctx, k))

	_, has = k.GetOrder(ctx, 1)
	require.False(t, has)
}

func TestOrders26(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "ukusd", keepertest.Dave, 10_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "5000",
		MaxPrice:       "0.5",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)

	require.Equal(t, order.AmountLocked.Int64(), int64(2_516))
	require.NoError(t, executeOrders(ctx, k))

	_, has = k.GetOrder(ctx, 1)
	require.False(t, has)
}

func TestOrders27(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "ukusd", keepertest.Dave, 10_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "10000",
		MaxPrice:       "0.5",
		IsBuyOrder:     true,
	}))

	require.NoError(t, executeOrders(ctx, k))
}

func TestOrders28(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 10_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 10_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, "ukusd", keepertest.Dave, 200_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "200000",
		MaxPrice:       "0.5",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(100604), order.AmountLocked.Int64())

	require.NoError(t, executeOrders(ctx, k))

	order, has = k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(97_260), order.AmountLocked.Int64())
	require.Equal(t, int64(200_000), order.AmountRequested.Int64())
	require.Equal(t, int64(9_950), order.AmountReceived.Int64())
	require.Equal(t, int64(3_344), order.AmountGiven.Int64())
}

func TestOrders31(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Dave, 20_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
		MaxPrice:       "0.2",
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(10_000), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)

	require.True(t, fullyExecuted)
	require.Equal(t, int64(10_000), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(2_473), tradeResult.AmountReceived.Int64())

	pricePaid, err := tradeResult.PricePaid()
	require.NoError(t, err)
	require.True(t, math.LegacyOneDec().Quo(pricePaid).LT(math.LegacyNewDec(5))) // C
}

func TestOrders32(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 20_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "10000",
		MaxPrice:       "3.5",
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(10_000), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)

	require.True(t, fullyExecuted)
	require.Equal(t, int64(10_000), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(39_288), tradeResult.AmountReceived.Int64())

	pricePaid, err := tradeResult.PricePaid()
	require.NoError(t, err)
	require.True(t, math.LegacyOneDec().Quo(pricePaid).GT(math.LegacyNewDecWithPrec(35, 1))) // C
}

func TestOrders33(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 20_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "40000",
		MaxPrice:       "0.3",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(12_073), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)
	require.False(t, tradeResult.AmountGiven.IsNil())
	require.False(t, tradeResult.AmountReceived.IsNil())

	require.True(t, fullyExecuted)
	require.Equal(t, int64(10_183), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(40_000), tradeResult.AmountReceived.Int64())

	pricePaid, err := tradeResult.PricePaid()
	require.NoError(t, err)
	require.True(t, pricePaid.LT(math.LegacyNewDecWithPrec(3, 1)))
}

func TestOrders34(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Dave, 60_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
		MaxPrice:       "5",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(50_302), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)
	require.False(t, tradeResult.AmountGiven.IsNil())
	require.False(t, tradeResult.AmountReceived.IsNil())

	require.True(t, fullyExecuted)
	require.Equal(t, int64(40_729), tradeResult.AmountGiven.Int64())
	require.Equal(t, int64(10_000), tradeResult.AmountReceived.Int64())

	pricePaid, err := tradeResult.PricePaid()
	require.NoError(t, err)
	require.True(t, pricePaid.LT(math.LegacyNewDec(5)))
}

func TestOrders35(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Dave, 60_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
		MaxPrice:       "5",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(50_302), order.AmountLocked.Int64())

	accOrders := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	balance := k.BankKeeper.SpendableCoins(ctx, accOrders.GetAddress())
	require.Equal(t, int64(50_302), balance.AmountOf(constants.BaseCurrency).Int64())

	require.NoError(t, executeOrders(ctx, k))

	balance = k.BankKeeper.SpendableCoins(ctx, accOrders.GetAddress())
	require.Equal(t, int64(0), balance.AmountOf(constants.BaseCurrency).Int64())
}

func TestOrders36(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.BaseCurrency, keepertest.Dave, 60_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.BaseCurrency,
		DenomReceiving: constants.KUSD,
		Amount:         "10000",
		MaxPrice:       "1",
		IsBuyOrder:     true,
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(10_061), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)

	require.False(t, fullyExecuted)
	require.True(t, tradeResult.AmountGiven.IsNil())
	require.True(t, tradeResult.AmountReceived.IsNil())
}

func TestOrders37(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 20_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: constants.BaseCurrency,
		Amount:         "10000",
		MaxPrice:       "5",
	}))

	order, has := k.GetOrder(ctx, 1)
	require.True(t, has)
	require.Equal(t, int64(10_000), order.AmountLocked.Int64())

	tradeResult, fullyExecuted, err := executeOrder(ctx, k, &order)
	require.NoError(t, err)

	require.False(t, fullyExecuted)
	require.True(t, tradeResult.AmountGiven.IsNil())
	require.True(t, tradeResult.AmountReceived.IsNil())
}

func TestOrders38(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.BaseCurrency, 1_000_000))
	require.NoError(t, keepertest.AddLiquidity(ctx, msg, keepertest.Alice, constants.KUSD, 1_000_000))

	keepertest.AddFunds(ctx, t, k.BankKeeper, constants.KUSD, keepertest.Dave, 2_000_000)

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "10000",
		MaxPrice:       "1.01",
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "10000",
		MaxPrice:       "1.02",
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "10000",
		MaxPrice:       "0.99",
		IsBuyOrder:     true,
	}))

	require.NoError(t, keepertest.AddOrder(ctx, msg, &types.MsgAddOrder{
		Creator:        keepertest.Dave,
		DenomGiving:    constants.KUSD,
		DenomReceiving: "uwusdc",
		Amount:         "10000",
		MaxPrice:       "0.98",
		IsBuyOrder:     true,
	}))

	o1, _ := k.GetOrder(ctx, 1)
	require.Equal(t, "1.010000000000000000", o1.MaxPrice.String())
	o2, _ := k.GetOrder(ctx, 2)
	require.Equal(t, "1.020000000000000000", o2.MaxPrice.String())
	o3, _ := k.GetOrder(ctx, 3)
	require.Equal(t, "0.990000000000000000", o3.MaxPrice.String())
	o4, _ := k.GetOrder(ctx, 4)
	require.Equal(t, "0.980000000000000000", o4.MaxPrice.String())

	require.NoError(t, executeOrders(ctx, k))
}

func randomAmount(max int) int {
	return rand.Intn(max-1) + 1
}

func checkOrderPoolBalanced(k dexkeeper.Keeper, ctx context.Context) bool {
	orderCoins := k.OrderSum(ctx)

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	coins := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		has, coin := coins.Find(denom)

		var poolAmount, sumOrder math.Int

		if has {
			poolAmount = coin.Amount
		} else {
			poolAmount = math.ZeroInt()
		}

		if orderSum, exists := orderCoins[denom]; exists {
			sumOrder = orderSum
		} else {
			sumOrder = math.ZeroInt()
		}

		diff := sumOrder.Sub(poolAmount).Abs().Int64()
		if diff > 1 {
			fmt.Println(fmt.Sprintf("%v vs %v", sumOrder.String(), poolAmount.String()))
			return false
		}
	}

	return true
}

func checkOrderPoolBalanceDiff(t *testing.T, k dexkeeper.Keeper, ctx context.Context) {
	orderCoins := k.OrderSum(ctx)

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	coins := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		has, coin := coins.Find(denom)

		var poolAmount int64
		if has {
			poolAmount = coin.Amount.Int64()
		}

		var sumOrder int64
		if orderSum, exists := orderCoins[denom]; exists {
			sumOrder = orderSum.Int64()
		}

		diff := poolAmount - sumOrder
		if diff != 0 {
			k.OrderSum(ctx)
		}

		if diff != 0 {
			fmt.Println(fmt.Sprintf("denom:%v, poolAmount: %v, sumOrder: %v", denom, poolAmount, sumOrder))
		}

		require.Zero(t, diff)
	}
}
