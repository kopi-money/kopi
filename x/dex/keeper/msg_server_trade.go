package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TradeFunc func(types.TradeContext) (types.TradeResult, error)

func (k msgServer) Sell(ctx context.Context, msg *types.MsgSell) (*types.MsgTradeResponse, error) {
	return k.trade(ctx, msg.Creator, msg.DenomGiving, msg.DenomReceiving, msg.Amount, msg.MaxPrice, msg.MinimumTradeAmount, k.ExecuteSell)
}

func (k msgServer) Buy(ctx context.Context, msg *types.MsgBuy) (*types.MsgTradeResponse, error) {
	return k.trade(ctx, msg.Creator, msg.DenomGiving, msg.DenomReceiving, msg.Amount, msg.MaxPrice, msg.MinimumTradeAmount, k.ExecuteBuy)
}

func (k msgServer) trade(ctx context.Context, creator, denomGiving, denomReceiving, amountString, maxPriceString, minimumTradeAmountString string, tradeFunc TradeFunc) (*types.MsgTradeResponse, error) {
	if denomGiving == denomReceiving {
		return nil, types.ErrSameDenom
	}

	tradeAmount, err := ParseAmount(amountString)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	if tradeAmount.IsZero() {
		return nil, types.ErrZeroAmount
	}

	maxPrice, err := stringToDec(maxPriceString)
	if err != nil {
		return nil, err
	}

	minimumTradeAmount, err := stringToInt(minimumTradeAmountString)
	if err != nil {
		return nil, err
	}

	if minimumTradeAmount != nil && !minimumTradeAmount.IsNil() && minimumTradeAmount.GT(tradeAmount) {
		return nil, types.ErrMinimumTradeAmountTooLarge
	}

	address, _ := sdk.AccAddressFromBech32(creator)
	tradeFee := k.GetParams(ctx).TradeFee

	tradeCtx := types.TradeContext{
		Context:                ctx,
		CoinSource:             creator,
		CoinTarget:             creator,
		TradeAmount:            tradeAmount,
		MaximumAvailableAmount: k.BankKeeper.SpendableCoin(ctx, address, denomGiving).Amount,
		MaxPrice:               maxPrice,
		MinimumTradeAmount:     minimumTradeAmount,
		TradeDenomGiving:       denomGiving,
		TradeDenomReceiving:    denomReceiving,
		ProtocolTrade:          false,
		TradeBalances:          NewTradeBalances(),
		Fee:                    k.getTradeFee(ctx, tradeFee, creator, denomGiving, denomReceiving, false),
	}

	tradeResult, err := tradeFunc(tradeCtx)
	if err != nil {
		return nil, fmt.Errorf("could not execute trade: %w", err)
	}

	if err = tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return nil, fmt.Errorf("could not settle balances: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("trade_executed",
			sdk.Attribute{Key: "address", Value: tradeCtx.CoinTarget},
			sdk.Attribute{Key: "denom_giving", Value: tradeCtx.TradeDenomGiving},
			sdk.Attribute{Key: "denom_receiving", Value: tradeCtx.TradeDenomReceiving},
			sdk.Attribute{Key: "amount_intermediate_base_currency", Value: tradeResult.AmountIntermediate.String()},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven.String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived.String()},
			sdk.Attribute{Key: "protocol_trade", Value: strconv.FormatBool(tradeCtx.ProtocolTrade)},
		),
	)

	return &types.MsgTradeResponse{
		AmountGiven:    tradeResult.AmountGiven.String(),
		AmountReceived: tradeResult.AmountReceived.String(),
	}, nil
}

func (k Keeper) getTradeFee(ctx context.Context, fee math.LegacyDec, discountAddress, denomGiving, denomReceiving string, excludeFromDiscount bool) math.LegacyDec {
	discount := k.getTradeDiscount(ctx, discountAddress, excludeFromDiscount)
	discount = math.LegacyOneDec().Sub(discount)
	fee = fee.Mul(discount)

	if denomGiving != constants.BaseCurrency && denomReceiving != constants.BaseCurrency {
		fee = fee.Quo(math.LegacyNewDec(2)) // C
	}

	return fee
}

func stringToDec(decString string) (*math.LegacyDec, error) {
	if decString == "" {
		return nil, nil
	}

	decString = strings.ReplaceAll(decString, ",", "")
	dec, err := math.LegacyNewDecFromStr(decString)
	if err != nil {
		return nil, types.ErrInvalidDecimalFormat
	}

	return &dec, nil
}

func stringToInt(intString string) (*math.Int, error) {
	if intString == "" {
		return nil, nil
	}

	intString = strings.ReplaceAll(intString, ",", "")
	integer, ok := math.NewIntFromString(intString)
	if !ok {
		return nil, types.ErrInvalidIntegerFormat
	}

	return &integer, nil
}
