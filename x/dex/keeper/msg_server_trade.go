package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/kopi-money/kopi/trading"

	"github.com/kopi-money/kopi/x/dex/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TradeFunc func(types.TradeContext) (trading.TradeResult, error)

func (k msgServer) Sell(ctx context.Context, msg *types.MsgSell) (*types.MsgTradeResponse, error) {
	return k.trade(ctx, msg.Creator, msg.DenomGiving, msg.DenomReceiving, msg.Amount, msg.MinimumTradeAmount, msg.MaxPrice, k.ExecuteSell)
}

func (k msgServer) Buy(ctx context.Context, msg *types.MsgBuy) (*types.MsgTradeResponse, error) {
	return k.trade(ctx, msg.Creator, msg.DenomGiving, msg.DenomReceiving, msg.Amount, msg.MinimumTradeAmount, msg.MaxPrice, k.ExecuteBuy)
}

func (k msgServer) trade(ctx context.Context, creator, denomGiving, denomReceiving, amountString, minimumTradeAmountString string, maxPrice *types.MaxPrice, tradeFunc TradeFunc) (*types.MsgTradeResponse, error) {
	if denomGiving == denomReceiving {
		return nil, types.ErrSameDenom
	}

	tradeAmount, err := trading.ParseAmount(amountString)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	if tradeAmount.IsZero() {
		return nil, types.ErrZeroAmount
	}

	maxPriceDec, err := trading.ParseMaxPrice(maxPrice.GetMaxPrice(), maxPrice.GetFeeIncluded())
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
		MaxPrice:               maxPriceDec,
		TradeAmount:            tradeAmount,
		MinimumTradeAmount:     minimumTradeAmount,
		Fee:                    &tradeFee,
		CoinSource:             creator,
		CoinTarget:             creator,
		MaximumAvailableAmount: k.BankKeeper.SpendableCoin(ctx, address, denomGiving).Amount,
		TradeDenomGiving:       denomGiving,
		TradeDenomReceiving:    denomReceiving,
		TradeBalances:          NewTradeBalances(),
	}

	tradeResult, err := tradeFunc(tradeCtx)
	if err != nil {
		return nil, fmt.Errorf("execute trade: %w", err)
	}

	if err = tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return nil, fmt.Errorf("settle balances: %w", err)
	}

	return &types.MsgTradeResponse{
		AmountGiven:    tradeResult.AmountGiven().String(),
		AmountReceived: tradeResult.AmountReceived().String(),
	}, nil
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
