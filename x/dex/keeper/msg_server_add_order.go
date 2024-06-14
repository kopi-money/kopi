package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) AddOrder(goCtx context.Context, msg *types.MsgAddOrder) (*types.Order, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.DenomFrom == msg.DenomTo {
		return nil, types.ErrSameDenom
	}

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}

	if amount.LT(k.DenomKeeper.MinOrderSize(ctx, msg.DenomFrom)) {
		return nil, types.ErrOrderSizeTooSmall
	}

	if msg.TradeAmount == "" {
		msg.TradeAmount = "0"
	}

	tradeAmount, err := parseAmount(msg.TradeAmount)
	if err != nil {
		return nil, err
	}

	address, err := k.validateMsg(ctx, msg.Creator, msg.DenomFrom, amount)
	if err != nil {
		return nil, err
	}

	maxPrice, err := getMaxPrice(msg.MaxPrice)
	if err != nil {
		return nil, err
	}

	if msg.Interval < 1 {
		msg.Interval = 1
	}

	if maxPrice == nil || maxPrice.IsNil() {
		return nil, types.ErrMaxPriceNotSet
	}

	if maxPrice.LTE(math.LegacyZeroDec()) {
		return nil, types.ErrNegativePrice
	}

	coins := sdk.NewCoins(sdk.NewCoin(msg.DenomFrom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolOrders, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to module")
	}

	order := types.Order{
		Creator:           msg.Creator,
		DenomFrom:         msg.DenomFrom,
		DenomTo:           msg.DenomTo,
		AmountLeft:        amount,
		TradeAmount:       tradeAmount,
		MaxPrice:          *maxPrice,
		AddedAt:           ctx.BlockHeight(),
		NumBlocks:         msg.Blocks,
		ExecutionInterval: msg.Interval,
		AllowIncomplete:   msg.AllowIncomplete,
	}

	order.Index = k.SetOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("order_created",
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(order.Index))},
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom_from", Value: msg.DenomFrom},
			sdk.Attribute{Key: "denom_to", Value: msg.DenomTo},
			sdk.Attribute{Key: "max_price", Value: maxPrice.String()},
			sdk.Attribute{Key: "blocks", Value: strconv.Itoa(int(msg.Blocks))},
			sdk.Attribute{Key: "amount_given", Value: amount.String()},
			sdk.Attribute{Key: "interval", Value: strconv.Itoa(int(msg.Interval))},
			sdk.Attribute{Key: "allow_incomplete", Value: strconv.FormatBool(msg.AllowIncomplete)},
		),
	)

	return &order, nil
}
