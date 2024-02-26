package keeper_test

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"testing"
)

func TestOrders1(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1",
	})
	require.Error(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1",
		MaxPrice:  "abc",
	})
	require.Error(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1",
		MaxPrice:  "1",
	})
	require.NoError(t, err)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders2(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1000",
		MaxPrice:  "1",
		Blocks:    1000,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 0, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders3(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1000",
		MaxPrice:  "100",
		Blocks:    1000,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 0, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders4(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1",
		MaxPrice:  "0.1",
		Blocks:    10,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 1, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders5(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1",
		MaxPrice:  "0.1001",
		Blocks:    1000,
	})
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100000",
		MaxPrice:  "11",
		Blocks:    1000,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 1, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders7(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: "ukusd",
		DenomTo:   utils.BaseCurrency,
		Amount:    "1000",
		MaxPrice:  "10",
		Blocks:    1000,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 0, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders8(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1",
		MaxPrice:  "10",
		Blocks:    10,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	numOrders := len(k.GetAllOrders(ctx))
	require.Equal(t, 1, numOrders)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders9(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	_, err := msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1",
		MaxPrice:  "10",
	})
	require.NoError(t, err)

	counter := 0
	for _, order := range k.GetAllOrders(ctx) {
		if order.Creator == keepertest.Bob {
			counter += 1
		}
	}
	require.Equal(t, 1, counter)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders10(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	address, err := sdk.AccAddressFromBech32(keepertest.Bob)
	require.NoError(t, err)

	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukopi")

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1",
		MaxPrice:  "10",
		Blocks:    0,
	})
	require.NoError(t, err)

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukopi")
	kopi2 = kopi2.Add(math.NewInt(1))

	require.Equal(t, kopi1, kopi2)

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders11(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	address, err := sdk.AccAddressFromBech32(keepertest.Bob)
	require.NoError(t, err)

	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukopi")
	kusd1 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukusd")

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "1000",
		MaxPrice:  "1000",
		Blocks:    100,
	})
	require.NoError(t, err)

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukopi")
	require.True(t, kopi2.LT(kopi1))

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	kusd2 := getCoins(k.BankKeeper.SpendableCoins(ctx, address), "ukusd")
	require.True(t, kusd2.GT(kusd1))

	require.True(t, checkOrderPoolBalanced(k, ctx))
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
	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")

	res, err := msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100",
		MaxPrice:  "1000",
		Blocks:    100,
	})
	require.NoError(t, err)

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")
	require.True(t, kopi2.GT(kopi1))
	require.Equal(t, kopi2, math.NewInt(100))

	_, err = msg.RemoveOrder(ctx, &types.MsgRemoveOrder{
		Creator: keepertest.Bob,
		Index:   res.Index,
	})
	require.NoError(t, err)

	kopi3 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")
	require.Equal(t, kopi3, math.NewInt(0))

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders13(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, keepertest.Pow(2))
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", keepertest.Pow(2))
	require.NoError(t, err)

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders)
	kopi1 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")

	res, err := msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100",
		MaxPrice:  "1000",
		Blocks:    100,
	})
	require.NoError(t, err)

	kopi2 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")
	require.True(t, kopi2.GT(kopi1))
	require.Equal(t, kopi2, math.NewInt(100))

	_, err = msg.RemoveOrder(ctx, &types.MsgRemoveOrder{
		Creator: keepertest.Bob,
		Index:   res.Index,
	})
	require.NoError(t, err)

	kopi3 := getCoins(k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()), "ukopi")
	require.Equal(t, kopi3, math.NewInt(0))

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders14(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000)
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100_000",
		MaxPrice:  "1000",
		Blocks:    100,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders15(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000)
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:   keepertest.Bob,
		DenomFrom: utils.BaseCurrency,
		DenomTo:   "ukusd",
		Amount:    "100_000",
		MaxPrice:  "1000",
		Blocks:    100,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))

	require.True(t, checkOrderPoolBalanced(k, ctx))
}

func TestOrders16(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000)
	require.NoError(t, err)

	var biggestDiff int64

	for i := 0; i < 100; i++ {
		_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomFrom:       utils.BaseCurrency,
			DenomTo:         "ukusd",
			Amount:          strconv.Itoa(randomAmount(10000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		})
		require.NoError(t, err)

		_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomFrom:       "ukusd",
			DenomTo:         utils.BaseCurrency,
			Amount:          strconv.Itoa(randomAmount(10000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		})
		require.NoError(t, err)

		require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))
		checkOrderPoolBalanceDiff(k, ctx, &biggestDiff)
	}

	require.Equal(t, int64(0), biggestDiff)
}

func TestOrders17(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 100)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 100)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "uwusdc", 100)
	require.NoError(t, err)

	var biggestDiff int64

	for i := 0; i < 100; i++ {
		_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomFrom:       "uwusdc",
			DenomTo:         "ukusd",
			Amount:          strconv.Itoa(randomAmount(1000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		})
		require.NoError(t, err)

		_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
			Creator:         keepertest.Bob,
			DenomFrom:       "ukusd",
			DenomTo:         "uwusdc",
			Amount:          strconv.Itoa(randomAmount(1000)),
			MaxPrice:        "1000",
			Blocks:          100,
			AllowIncomplete: true,
		})
		require.NoError(t, err)

		require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))
		checkOrderPoolBalanceDiff(k, ctx, &biggestDiff)
	}

	require.Equal(t, int64(0), biggestDiff)
}

func TestOrders18(t *testing.T) {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	err := keepertest.AddLiquidity(ctx, msg, keepertest.Alice, utils.BaseCurrency, 1_000_000)
	require.NoError(t, err)
	err = keepertest.AddLiquidity(ctx, msg, keepertest.Alice, "ukusd", 1_000_000)
	require.NoError(t, err)

	_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
		Creator:         keepertest.Bob,
		DenomFrom:       utils.BaseCurrency,
		DenomTo:         "ukusd",
		Amount:          "10000",
		TradeAmount:     "5000",
		MaxPrice:        "1000",
		Blocks:          100,
		AllowIncomplete: true,
	})
	require.NoError(t, err)

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))
	require.Equal(t, 1, len(k.GetAllOrders(ctx)))
	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), ctx.BlockHeight()))
	require.Equal(t, 0, len(k.GetAllOrders(ctx)))
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

func checkOrderPoolBalanceDiff(k dexkeeper.Keeper, ctx context.Context, biggestDiff *int64) {
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
		if biggestDiff != nil && diff > *biggestDiff {
			*biggestDiff = diff
		}
	}
}
