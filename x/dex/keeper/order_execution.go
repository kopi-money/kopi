package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	"github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var skipErrors = []error{
	types.ErrTradeAmountTooSmall,
	types.ErrNotEnoughLiquidity,
	types.ErrPriceTooLow,
	types.ErrZeroTrade,
	types.ErrNegativeTradeAmount,

	constant_product.ErrRequestedAmountTooLarge,
}

func (k Keeper) ExecuteOrders(ctx context.Context, eventManager sdk.EventManagerI, blockHeight int64) error {
	ordersCaches := k.NewOrdersCaches(ctx)
	fee := k.GetJoinedFee(ctx)
	maxOrderLife := int64(k.GetParams(ctx).MaxOrderLife)
	iterator := k.OrderIterator(ctx)
	tradeBalances := NewTradeBalances()

	numTrades := 0
	tradeVolumeBaseSum := math.ZeroInt()

	// At this point we know that there are no changes in the ongoing transaction. To avoid the costly iteration over
	// two lists (of which the second is empty but needs to be checked at every step), we just get all the items from
	// cache directly
	orders := iterator.GetAllFromCache()

	for index, keyValue := range orders {
		order := keyValue.Value().Value()
		if order == nil {
			k.Logger().Info("order is nil, should not happen")
			continue
		}

		// First we check whether the order is expired. If yes, it is removed.
		blockEnd := order.AddedAt + min(maxOrderLife, int64(order.NumBlocks))
		if blockHeight > blockEnd {
			if !order.AmountLeft.IsNil() && order.AmountLeft.IsPositive() {
				tradeBalances.AddTransfer(ordersCaches.AccPoolOrders.Get().String(), order.Creator, order.DenomGiving, order.AmountLocked)
			}

			eventManager.EmitEvent(
				sdk.NewEvent("order_expired",
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
					sdk.Attribute{Key: "address", Value: order.Creator},
					sdk.Attribute{Key: "denom_giving", Value: order.DenomGiving},
					sdk.Attribute{Key: "denom_receiving", Value: order.DenomReceiving},
					sdk.Attribute{Key: "amount_given", Value: order.AmountGiven.String()},
					sdk.Attribute{Key: "amount_received", Value: order.AmountReceived.String()},
					sdk.Attribute{Key: "max_price", Value: order.MaxPrice.String()},
					sdk.Attribute{Key: "is_buy_order", Value: strconv.FormatBool(order.IsBuyOrder)},
				),
			)

			k.orders.Remove(ctx, order.Index)
			continue
		}

		// Next we check whether the order is to be executed at this height
		if (order.AddedAt+blockHeight)%int64(order.ExecutionInterval) != 0 {
			continue
		}

		// Next we do the actual execution
		tradeResult, remove, err := k.ExecuteOrder(ctx, ordersCaches, fee, order)
		if err != nil {
			return fmt.Errorf("executing order (%v / %v): %w", index, order.Index, err)
		}

		if !tradeResult.AmountIntermediate.IsNil() && tradeResult.AmountIntermediate.IsPositive() {
			tradeVolumeBaseSum = tradeVolumeBaseSum.Add(tradeResult.AmountIntermediate)
			numTrades++
		}

		if remove {
			eventManager.EmitEvent(
				sdk.NewEvent("order_completed",
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
					sdk.Attribute{Key: "address", Value: order.Creator},
					sdk.Attribute{Key: "denom_giving", Value: order.DenomGiving},
					sdk.Attribute{Key: "denom_receiving", Value: order.DenomReceiving},
					sdk.Attribute{Key: "amount_given", Value: order.AmountGiven.String()},
					sdk.Attribute{Key: "amount_received", Value: order.AmountReceived.String()},
					sdk.Attribute{Key: "max_price", Value: order.MaxPrice.String()},
					sdk.Attribute{Key: "is_buy_order", Value: strconv.FormatBool(order.IsBuyOrder)},
				),
			)

			if err = k.RemoveOrder(ctx, *order); err != nil {
				return fmt.Errorf("removing order: %w", err)
			}
		}
	}

	if numTrades > 0 {
		eventManager.EmitEvent(
			sdk.NewEvent("orders_executed",
				sdk.Attribute{Key: "num_trades", Value: strconv.Itoa(numTrades)},
				sdk.Attribute{Key: "amount_intermediate_base_currency", Value: tradeVolumeBaseSum.String()},
			),
		)
	}

	if err := tradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("settling trade balances: %w", err)
	}

	return nil
}

func (k Keeper) ExecuteOrder(ctx context.Context, ordersCaches *types.OrdersCaches, fee math.LegacyDec, order *types.Order) (types.TradeResult, bool, error) {
	// When the price of this order is "worse" than the ones of previously checked, we can skip all other checks. If the
	// order with the "better" price could not be executed, the one with the "worse" price cannot be executed as well.
	denomPair := types.Pair{DenomFrom: order.DenomGiving, DenomTo: order.DenomReceiving}
	if !ordersCaches.BetterThanPreviousPrice(denomPair, order.MaxPrice, order.IsBuyOrder) {
		return types.TradeResult{}, false, nil
	}

	if order.MaxPrice.IsNil() {
		k.Logger().Error(fmt.Sprintf("max_price for order %v is null", order.Index))
		return types.TradeResult{}, false, nil
	}

	if !order.MaxPrice.IsPositive() {
		k.Logger().Error(fmt.Sprintf("max_price for order %v is not positive", order.Index))
		return types.TradeResult{}, false, nil
	}

	maxPrice := order.MaxPrice
	if !order.IsBuyOrder {
		maxPrice = math.LegacyOneDec().Quo(maxPrice) // C
	}

	address := sdk.MustAccAddressFromBech32(order.Creator)
	orderTradeBalances := NewTradeBalances()
	tradeCtx := types.TradeContext{
		Context:                ctx,
		CoinSource:             ordersCaches.AccPoolOrders.Get().String(),
		CoinTarget:             address.String(),
		TradeAmount:            order.AmountLeft,
		MaximumAvailableAmount: order.AmountLocked,
		TradeDenomGiving:       order.DenomGiving,
		TradeDenomReceiving:    order.DenomReceiving,
		MaxPrice:               &maxPrice,
		TradeBalances:          orderTradeBalances,
		OrdersCaches:           ordersCaches,
		IsOrder:                true,
		Fee:                    fee,
	}

	if order.TradeAmount.IsPositive() {
		tradeCtx.TradeAmount = math.MinInt(tradeCtx.TradeAmount, order.TradeAmount)
	}

	tradeResult, err := k.getTradeFunction(order.IsBuyOrder)(tradeCtx)
	if err != nil {
		if errors.Is(err, types.ErrNegativeTradeAmount) {
			ordersCaches.SetPreviousPrice(denomPair, maxPrice, order.IsBuyOrder)
		}

		if isSkipError(err) {
			return types.TradeResult{}, false, nil
		}

		var msg string
		if tradeCtx.TradeType == types.TradeTypeSell {
			msg = fmt.Sprintf("execute trade (%v%v > %v)", tradeCtx.TradeAmount.String(), order.DenomGiving, order.DenomReceiving)
		} else {
			msg = fmt.Sprintf("execute trade (%v > %v%v)", order.DenomGiving, tradeCtx.TradeAmount.String(), order.DenomReceiving)
		}

		k.Logger().Error(fmt.Errorf("%v: %w", msg, err).Error())
		return types.TradeResult{}, false, nil
	}

	if err = orderTradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return types.TradeResult{}, false, fmt.Errorf("settling balances: %w", err)
	}

	if tradeResult.AmountGiven.IsZero() {
		return types.TradeResult{}, false, nil
	}

	order.AmountLocked = order.AmountLocked.Sub(tradeResult.AmountGiven)
	order.AmountGiven = order.AmountGiven.Add(tradeResult.AmountGiven)
	order.AmountReceived = order.AmountReceived.Add(tradeResult.AmountReceived)

	if order.IsBuyOrder {
		order.AmountLeft = order.AmountLeft.Sub(tradeResult.AmountReceived)
	} else {
		order.AmountLeft = order.AmountLeft.Sub(tradeResult.AmountGiven)
	}

	// AmountLeft and AmountLocked should never be negative zero. The comparison is still considering lower
	// than zero to cover potential rounding issues
	fullyExecuted := !order.AmountLeft.GTE(math.NewInt(constants.MinimumTradeSize)) || !order.AmountLocked.IsPositive()

	if order.AmountLeft.IsNegative() {
		return types.TradeResult{}, false, fmt.Errorf("order has negative amount left (%v, %v)", tradeResult.AmountGiven.String(), order.AmountLeft.String())
	}

	if !fullyExecuted {
		k.SetOrder(ctx, *order)
	}

	return tradeResult, fullyExecuted, nil
}

// calculateBlockEnd calculates the maximum block height that an order can be alive. If the requested block height is
// bigger than the time allowed by the parameter, the height is capped to the allowed limit.
func (k Keeper) calculateBlockEnd(maxOrderLife, addedAt, numBlocks int64) int64 {
	var life int64
	if numBlocks > maxOrderLife {
		life = maxOrderLife
	} else {
		life = numBlocks
	}

	return addedAt + life
}

func (k Keeper) getTradeFunction(isBuyOrder bool) func(ctx types.TradeContext) (types.TradeResult, error) {
	if isBuyOrder {
		return k.ExecuteBuy
	} else {
		return k.ExecuteSell
	}
}

func isSkipError(err error) bool {
	for _, skipError := range skipErrors {
		if errors.Is(err, skipError) {
			return true
		}
	}

	return false
}
