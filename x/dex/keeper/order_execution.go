package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/pkg/errors"
	"strconv"
)

func (k Keeper) ExecuteOrders(ctx context.Context, eventManager sdk.EventManagerI, blockHeight int64) error {
	iterator := k.OrdersIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order types.Order
		k.cdc.MustUnmarshal(iterator.Value(), &order)

		remove, err := k.executeOrder(ctx, eventManager, order)
		if err != nil {
			return errors.Wrap(err, "error executing order")
		}

		tooOld := blockHeight >= int64(order.BlockEnd)

		if remove || tooOld {
			if remove {
				eventManager.EmitEvent(
					sdk.NewEvent("order_completed",
						sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
					),
				)
			}

			if tooOld {
				if !order.AmountLeft.IsNil() && order.AmountLeft.GT(math.ZeroInt()) {
					coins := sdk.NewCoins(sdk.NewCoin(order.DenomFrom, order.AmountLeft))
					address, _ := sdk.AccAddressFromBech32(order.Creator)
					if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOrders, address, coins); err != nil {
						return errors.Wrap(err, "could not send left over coins from module to user")
					}
				}

				eventManager.EmitEvent(
					sdk.NewEvent("order_expired",
						sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
					),
				)
			}

			k.RemoveOrder(ctx, order)
		}
	}

	return nil
}

func (k Keeper) executeOrder(ctx context.Context, eventManager sdk.EventManagerI, order types.Order) (bool, error) {
	fee := k.GetTradeFee(ctx)
	priceAmount := k.calculateAmountGivenPrice(ctx, order.DenomFrom, order.DenomTo, order.MaxPrice, fee).TruncateInt()
	if priceAmount.LTE(math.ZeroInt()) {
		return false, nil
	}

	amount := math.MinInt(order.AmountLeft, priceAmount)
	if amount.LTE(math.ZeroInt()) {
		return false, nil
	}

	if order.TradeAmount.GT(math.ZeroInt()) {
		amount = math.MinInt(amount, order.TradeAmount)
	}

	address := sdk.MustAccAddressFromBech32(order.Creator)
	poolAddress := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders).GetAddress()

	options := types.TradeOptions{
		CoinSource:      poolAddress,
		CoinTarget:      address,
		GivenAmount:     amount,
		TradeDenomStart: order.DenomFrom,
		TradeDenomEnd:   order.DenomTo,
		MaxPrice:        &order.MaxPrice,
		AllowIncomplete: order.AllowIncomplete,
	}

	usedAmount, receivedAmount, _, _, err := k.ExecuteTrade(ctx, eventManager, options)
	if err != nil {
		if errors.Is(err, types.ErrTradeAmountTooSmall) {
			return false, nil
		}
		if errors.Is(err, types.ErrNotEnoughLiquidity) {
			return false, nil
		}

		msg := fmt.Sprintf("could not execute trade (%v%v > %v)", order.AmountLeft.String(), order.DenomFrom, order.DenomTo)
		return false, errors.Wrap(err, msg)
	}

	if usedAmount.Equal(math.ZeroInt()) {
		return false, nil
	}

	order.AmountLeft = order.AmountLeft.Sub(usedAmount)
	order.AmountReceived = order.AmountReceived.Add(receivedAmount)

	if order.AmountLeft.LT(math.ZeroInt()) {
		return false, fmt.Errorf("order has negative amount left (%v, %v)", usedAmount.String(), order.AmountLeft.String())
	}

	k.SetOrder(ctx, order)

	eventManager.EmitEvent(
		sdk.NewEvent("order_executed",
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
			sdk.Attribute{Key: "amount_used", Value: usedAmount.String()},
			sdk.Attribute{Key: "amount_received", Value: receivedAmount.String()},
		),
	)

	// AmountLeft should never be negative zero. The comparison is still considering lower
	// than zero to cover potential rounding issues
	fullyExecuted := order.AmountLeft.LTE(math.ZeroInt())
	return fullyExecuted, nil
}

func (k Keeper) checkOrderPoolBalanced(ctx context.Context) bool {
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
			sumOrder = math.LegacyZeroDec().RoundInt()
		}

		if sumOrder.Sub(poolAmount).Abs().GT(math.OneInt()) {
			fmt.Println(fmt.Sprintf("%v vs %v", sumOrder.String(), poolAmount.String()))
			return false
		}
	}

	return true
}
