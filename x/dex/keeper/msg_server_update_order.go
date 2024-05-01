package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateOrder(goCtx context.Context, msg *types.MsgUpdateOrder) (*types.Order, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := k.GetOrder(ctx, msg.Index)
	if !found {
		return nil, types.ErrOrderNotFound
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if order.Creator != msg.Creator {
		return nil, types.ErrInvalidCreator
	}

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse amount")
	}

	if msg.TradeAmount == "" {
		msg.TradeAmount = "0"
	}

	tradeAmount, err := parseAmount(msg.TradeAmount)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse trade amount")
	}

	coins := sdk.NewCoins(sdk.NewCoin(order.DenomFrom, order.AmountLeft))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOrders, address, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to address")
	}

	if err = k.checkSpendableCoins(ctx, address, order.DenomFrom, amount); err != nil {
		return nil, errors.Wrap(err, "could not check spendable coins")
	}

	coins = sdk.NewCoins(sdk.NewCoin(order.DenomFrom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolOrders, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to module")
	}

	maxPrice, err := getMaxPrice(msg.MaxPrice)
	if err != nil {
		return nil, errors.Wrap(err, "could not get max price")
	}

	if maxPrice == nil {
		maxPrice = &order.MaxPrice
	}

	amountChange := amount.Sub(order.AmountLeft)

	order.AmountLeft = amount
	order.TradeAmount = tradeAmount
	order.MaxPrice = *maxPrice

	k.SetOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("order_updated",
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
			sdk.Attribute{Key: "amount_changed", Value: amountChange.String()},
			sdk.Attribute{Key: "max_price", Value: maxPrice.String()},
		),
	)

	return &order, nil
}
