package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) AddOrder(ctx context.Context, msg *types.MsgAddOrder) (*types.Order, error) {
	return k.Keeper.AddOrder(ctx, msg.Creator, msg.DenomGiving, msg.DenomReceiving, msg.Amount, msg.MaxPrice, msg.TradeAmount, msg.Interval, msg.Blocks, msg.IsBuyOrder)
}

func (k Keeper) AddOrder(ctx context.Context, creator, denomGiving, denomReceiving, amountString, maxPriceString, tradeAmountString string, interval, numBlocks uint64, isBuyOrder bool) (*types.Order, error) {
	if denomGiving == denomReceiving {
		return nil, types.ErrSameDenom
	}

	amount, err := ParseAmount(amountString)
	if err != nil {
		return nil, err
	}

	if isBuyOrder && amount.LT(k.DenomKeeper.MinOrderSize(ctx, denomReceiving)) {
		return nil, types.ErrOrderSizeTooSmall
	}

	if !isBuyOrder && amount.LT(k.DenomKeeper.MinOrderSize(ctx, denomGiving)) {
		return nil, types.ErrOrderSizeTooSmall
	}

	maxPrice, err := stringToDec(maxPriceString)
	if err != nil {
		return nil, err
	}

	if maxPrice == nil || maxPrice.IsNil() {
		return nil, types.ErrMaxPriceNotSet
	}

	if maxPrice.IsZero() {
		return nil, types.ErrZeroPrice
	}

	if maxPrice.IsNegative() {
		return nil, types.ErrNegativePrice
	}

	var lockAmount math.Int
	if isBuyOrder {
		lockAmount = k.calculateBuyLockAmount(ctx, amount.ToLegacyDec(), *maxPrice)
	} else {
		lockAmount = amount
	}

	if err = k.precheckTrade(ctx, creator, denomGiving, &lockAmount, false); err != nil {
		return nil, err
	}

	if tradeAmountString == "" {
		tradeAmountString = "0"
	}

	tradeAmount, err := ParseAmount(tradeAmountString)
	if err != nil {
		return nil, err
	}

	acc, _ := sdk.AccAddressFromBech32(creator)
	coins := sdk.NewCoins(sdk.NewCoin(denomGiving, lockAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolOrders, coins); err != nil {
		return nil, fmt.Errorf("send coins to module: %w", err)
	}

	order := types.Order{
		Creator:           creator,
		DenomGiving:       denomGiving,
		DenomReceiving:    denomReceiving,
		AmountRequested:   amount,
		AmountLeft:        amount,
		AmountLocked:      lockAmount,
		AmountGiven:       math.ZeroInt(),
		AmountReceived:    math.ZeroInt(),
		TradeAmount:       tradeAmount,
		MaxPrice:          *maxPrice,
		AddedAt:           sdk.UnwrapSDKContext(ctx).BlockHeight(),
		NumBlocks:         max(1, numBlocks),
		ExecutionInterval: max(1, interval),
		AllowIncomplete:   true,
		IsBuyOrder:        isBuyOrder,
	}

	order.Index = k.SetOrder(ctx, order)

	return &order, nil
}

func (k Keeper) calculateBuyLockAmount(ctx context.Context, amountRequested, maxPrice math.LegacyDec) math.Int {
	amountRequired := amountRequested.Mul(maxPrice)
	amountRequired = amountRequired.Quo(math.LegacyOneDec().Sub(k.GetOrderFee(ctx))) // C
	amountRequired = amountRequired.Quo(math.LegacyOneDec().Sub(k.GetTradeFee(ctx))) // C

	return amountRequired.Ceil().TruncateInt()
}

func (k msgServer) RemoveOrder(ctx context.Context, msg *types.MsgRemoveOrder) (*types.Void, error) {
	order, found := k.GetOrder(ctx, msg.Index)
	if !found {
		return nil, types.ErrItemNotFound
	}

	if order.Creator != msg.Creator {
		return nil, types.ErrInvalidCreator
	}

	if err := k.Keeper.RemoveOrder(ctx, order); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) RemoveOrders(ctx context.Context, msg *types.MsgRemoveOrders) (*types.Void, error) {
	for _, order := range k.GetAllOrdersByAddress(ctx, msg.Creator) {
		if err := k.Keeper.RemoveOrder(ctx, order); err != nil {
			return nil, err
		}
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateOrder(ctx context.Context, msg *types.MsgUpdateOrder) (*types.Order, error) {
	order, found := k.GetOrder(ctx, msg.Index)
	if !found {
		return nil, types.ErrOrderNotFound
	}

	if order.Creator != msg.Creator {
		return nil, types.ErrInvalidCreator
	}

	if err := k.Keeper.RemoveOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("remove order: %w", err)
	}

	newOrder, err := k.Keeper.AddOrder(ctx, msg.Creator, order.DenomGiving, order.DenomReceiving, msg.Amount, msg.MaxPrice, msg.TradeAmount, order.ExecutionInterval, order.NumBlocks, order.IsBuyOrder)
	if err != nil {
		return nil, fmt.Errorf("add order: %w", err)
	}

	return newOrder, nil
}
