package keeper_test

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/testutil/testdata"
	"math/rand"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/utils"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
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

//func TestOrders19(t *testing.T) {
//	orders1 := testOrdersFromData(t, false)
//	orders2 := testOrdersFromData(t, true)
//	require.True(t, compareOrderLists(orders1, orders2))
//}

func testOrdersFromData(t *testing.T, useLM bool) []types.Order {
	k, msg, ctx := keepertest.SetupDexMsgServer(t)

	liquidity, err := testdata.LoadLiquidity()
	require.NoError(t, err)
	orders, err := testdata.LoadOrders()
	require.NoError(t, err)

	fmt.Println(len(liquidity), len(orders))

	funds := make(map[string]Funds)
	gatherFundsFromLiquidity(t, funds, liquidity)
	gatherFundsFromOrders(t, funds, orders)

	for address, addressFunds := range funds {
		for denom, amount := range addressFunds {
			keepertest.AddFunds(t, ctx, k.BankKeeper, denom, address, amount)
		}
	}

	for _, order := range orders {
		_, err = msg.AddOrder(ctx, &types.MsgAddOrder{
			Creator:         order.Creator,
			DenomFrom:       order.DenomFrom,
			DenomTo:         order.DenomTo,
			Amount:          order.AmountLeft,
			TradeAmount:     order.TradeAmount,
			MaxPrice:        order.MaxPrice,
			Blocks:          100,
			Interval:        1,
			AllowIncomplete: order.AllowIncomplete,
		})

		require.NoError(t, err)
	}

	for _, liq := range liquidity {
		_, err = msg.AddLiquidity(ctx, &types.MsgAddLiquidity{
			Creator: liq.Address,
			Denom:   liq.Denom,
			Amount:  liq.Amount,
		})
		require.NoError(t, err)
	}

	require.NoError(t, k.ExecuteOrders(ctx, ctx.EventManager(), 0))
	return k.GetAllOrders(ctx)
}

func compareOrderLists(orders1, orders2 []types.Order) bool {
	if len(orders1) != len(orders2) {
		return false
	}

	for index := range len(orders1) {
		o1 := orders1[index]
		o2 := orders2[index]

		if o1.Index != o2.Index {
			return false
		}

		if o1.Creator != o2.Creator {
			return false
		}

		if !o1.AmountLeft.Equal(o2.AmountLeft) {
			return false
		}
	}

	return true
}

type Funds map[string]int64

func (f Funds) add(denom string, amount int64) {
	f[denom] += amount
}

func gatherFundsFromLiquidity(t *testing.T, funds map[string]Funds, liquidityEntries []testdata.LiquidityEntry) {
	for _, liq := range liquidityEntries {
		f, has := funds[liq.Address]
		if !has {
			funds[liq.Address] = make(Funds)
			f = funds[liq.Address]
		}

		amount, err := strconv.Atoi(liq.Amount)
		require.NoError(t, err)
		f.add(liq.Denom, int64(amount))
	}
}

func gatherFundsFromOrders(t *testing.T, funds map[string]Funds, orders []testdata.Order) {
	for _, order := range orders {
		f, has := funds[order.Creator]
		if !has {
			funds[order.Creator] = make(Funds)
			f = funds[order.Creator]
		}

		amount, err := strconv.Atoi(order.AmountLeft)
		require.NoError(t, err)
		f.add(order.DenomFrom, int64(amount))
	}
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
