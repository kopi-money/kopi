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
	ordersCaches := k.NewOrdersCaches(ctx)
	poolAddress := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOrders).GetAddress()
	fee := k.GetTradeFee(ctx)
	iterator := k.OrderIterator(ctx)

	for iterator.Valid() {
		order := iterator.GetNext()
		blockEnd := k.calculateBlockEnd(ctx, blockHeight, int64(order.NumBlocks))
		if blockHeight > blockEnd {
			if !order.AmountLeft.IsNil() && order.AmountLeft.GT(math.ZeroInt()) {
				coins := sdk.NewCoins(sdk.NewCoin(order.DenomFrom, order.AmountLeft))
				address, _ := sdk.AccAddressFromBech32(order.Creator)
				if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOrders, address, coins); err != nil {
					return errors.Wrap(err, "could not send left over coins from module to user")
				}
			}

			eventManager.EmitEvent(
				sdk.NewEvent("order_expired",
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
				),
			)

			k.RemoveOrder(ctx, order)
		}

		if (order.AddedAt+blockHeight)%int64(order.ExecutionInterval) != 0 {
			continue
		}

		remove, err := k.executeOrder(ctx, eventManager, ordersCaches, poolAddress, fee, &order)
		if err != nil {
			return errors.Wrap(err, "error executing order")
		}

		if remove {
			eventManager.EmitEvent(
				sdk.NewEvent("order_completed",
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
				),
			)

			k.RemoveOrder(ctx, order)
		}
	}

	return nil
}

func (k Keeper) executeOrder(ctx context.Context, eventManager sdk.EventManagerI, ordersCaches *types.OrdersCaches, poolAddress sdk.AccAddress, fee math.LegacyDec, order *types.Order) (bool, error) {
	denomPair := types.Pair{DenomFrom: order.DenomFrom, DenomTo: order.DenomTo}
	previousMaxPrice, has := ordersCaches.PriceAmounts[denomPair]
	if has && order.MaxPrice.LT(previousMaxPrice) {
		return false, nil
	}

	priceAmount := k.calculateAmountGivenPrice(ordersCaches, order.DenomFrom, order.DenomTo, order.MaxPrice, fee).TruncateInt()
	if priceAmount.LTE(math.ZeroInt()) {
		if !has || previousMaxPrice.LT(order.MaxPrice) {
			ordersCaches.PriceAmounts[denomPair] = order.MaxPrice
		}

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
	options := types.TradeOptions{
		CoinSource:      poolAddress,
		CoinTarget:      address,
		GivenAmount:     amount,
		TradeDenomStart: order.DenomFrom,
		TradeDenomEnd:   order.DenomTo,
		MaxPrice:        &order.MaxPrice,
		AllowIncomplete: order.AllowIncomplete,
		OrdersCaches:    ordersCaches,
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

	if order.AmountLeft.LT(math.ZeroInt()) {
		return false, fmt.Errorf("order has negative amount left (%v, %v)", usedAmount.String(), order.AmountLeft.String())
	}

	k.SetOrder(ctx, *order)

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

// calculateBlockEnd calculates the maximum block height that an order can be alive. If the requested block height is
// bigger than the time allowed by the parameter, the height is capped to the allowed limit.
func (k Keeper) calculateBlockEnd(ctx context.Context, addedAt, numBlocks int64) int64 {
	maxLife := int64(k.GetParams(ctx).MaxOrderLife)

	var life int64
	if numBlocks > maxLife {
		life = maxLife
	} else {
		life = numBlocks
	}

	return addedAt + life
}
