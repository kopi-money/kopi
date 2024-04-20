package keeper

import (
	"context"
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
	"strings"
)

func (k msgServer) Trade(goCtx context.Context, msg *types.MsgTrade) (*types.MsgTradeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.DenomFrom == msg.DenomTo {
		return nil, types.ErrSameDenom
	}

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not parse amount")
	}

	if amount.Equal(math.ZeroInt()) {
		return nil, types.ErrZeroAmount
	}

	address, err := k.validateMsg(ctx, msg.Creator, msg.DenomFrom, amount)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "invalid message")
	}

	maxPrice, err := getMaxPrice(msg.MaxPrice)
	if err != nil {
		return nil, err
	}

	options := types.TradeOptions{
		CoinSource:      address,
		CoinTarget:      address,
		GivenAmount:     amount,
		MaxPrice:        maxPrice,
		TradeDenomStart: msg.DenomFrom,
		TradeDenomEnd:   msg.DenomTo,
		AllowIncomplete: msg.AllowIncomplete,
	}

	amountUsed, amountReceived, _, _, err := k.ExecuteTrade(ctx, ctx.EventManager(), options)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not execute trade")
	}

	response := types.MsgTradeResponse{
		AmountReceived: amountReceived.Int64(),
		AmountUsed:     amountUsed.Int64(),
	}

	return &response, nil
}

func (k Keeper) getTradeFee(ctx context.Context, denomFrom, denomTo, address string, excludeFromDiscount bool) math.LegacyDec {
	if excludeFromDiscount {
		return math.LegacyZeroDec()
	}

	if address == "" {
		return math.LegacyZeroDec()
	}

	// Users have to pay fee for every step of a trade. However, when the trade consists of two steps, they only have
	// to pay half fee for each step.
	fee := k.GetTradeFee(ctx)
	if denomFrom != utils.BaseCurrency && denomTo != utils.BaseCurrency {
		fee = fee.Quo(math.LegacyNewDec(2))
	}

	discount := k.getTradeDiscount(ctx, address)
	discount = math.LegacyOneDec().Sub(discount)
	fee = fee.Mul(discount)

	return fee
}

func getMaxPrice(maxPriceString string) (*math.LegacyDec, error) {
	if maxPriceString == "" {
		return nil, nil
	}

	maxPriceString = strings.ReplaceAll(maxPriceString, ",", "")
	maxPrice, err := math.LegacyNewDecFromStr(maxPriceString)
	if err != nil {
		return nil, types.ErrInvalidPriceFormat
	}

	return &maxPrice, nil
}
