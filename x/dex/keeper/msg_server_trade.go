package keeper

import (
	"context"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
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

	tradeCtx := types.TradeContext{
		Context:         goCtx,
		CoinSource:      address.String(),
		CoinTarget:      address.String(),
		GivenAmount:     amount,
		MaxPrice:        maxPrice,
		TradeDenomStart: msg.DenomFrom,
		TradeDenomEnd:   msg.DenomTo,
		AllowIncomplete: msg.AllowIncomplete,
		ProtocolTrade:   false,
	}

	amountUsed, amountReceived1, amountReceived2, _, _, err := k.ExecuteTrade(tradeCtx)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not execute trade")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: tradeCtx.CoinTarget},
			sdk.Attribute{Key: "from", Value: tradeCtx.TradeDenomStart},
			sdk.Attribute{Key: "to", Value: tradeCtx.TradeDenomEnd},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: amountReceived1.String()},
			sdk.Attribute{Key: "amount_used", Value: amountUsed.String()},
			sdk.Attribute{Key: "amount_received", Value: amountReceived2.String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(tradeCtx.ProtocolTrade)},
		),
	)

	response := types.MsgTradeResponse{
		AmountReceived: amountReceived2.Int64(),
		AmountUsed:     amountUsed.Int64(),
	}

	return &response, nil
}

func (k Keeper) getTradeFee(ctx types.TradeContext) math.LegacyDec {
	// Users have to pay fee for every step of a trade. However, when the trade consists of two steps, they only have
	// to pay half fee for each step.
	fee := k.GetTradeFee(ctx)
	if ctx.TradeDenomStart != utils.BaseCurrency && ctx.TradeDenomEnd != utils.BaseCurrency {
		fee = fee.Quo(math.LegacyNewDec(2))
	}

	discount := k.getTradeDiscount(ctx, ctx.DiscountAddress, ctx.ExcludeFromDiscount)
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
