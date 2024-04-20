package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"strconv"

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
		AmountGiven:       amount,
		AmountReceived:    math.ZeroInt(),
		TradeAmount:       tradeAmount,
		MaxPrice:          *maxPrice,
		NumBlocks:         msg.Blocks,
		ExecutionInterval: msg.Interval,
		NextExecution:     uint64(ctx.BlockHeight()),
		BlockEnd:          k.calculateBlockEnd(ctx, msg.Blocks, uint64(ctx.BlockHeight())),
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

// calculateBlockEnd calculates the maximum block height that an order can be alive. If the requested block height is
// bigger than the time allowed by the parameter, the height is capped to the allowed limit.
func (k Keeper) calculateBlockEnd(ctx context.Context, blocks, blockHeight uint64) uint64 {
	maxLife := k.GetParams(ctx).MaxOrderLife

	var life uint64
	if blocks > maxLife {
		life = maxLife
	} else {
		life = blocks
	}

	return life + blockHeight
}
