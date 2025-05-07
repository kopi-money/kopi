package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type simulate func(ctx types.TradeContext) (types.TradeSimulationResult, error)

func (k Keeper) QuerySimulateSell(ctx context.Context, req *types.QuerySimulateTradeRequest) (*types.QuerySimulateTradeResponse, error) {
	return k.querySimulateTrade(ctx, req, k.SimulateSell)
}

func (k Keeper) QuerySimulateBuy(ctx context.Context, req *types.QuerySimulateTradeRequest) (*types.QuerySimulateTradeResponse, error) {
	return k.querySimulateTrade(ctx, req, k.SimulateBuy)
}

func (k Keeper) querySimulateTrade(ctx context.Context, req *types.QuerySimulateTradeRequest, simulateFunc simulate) (*types.QuerySimulateTradeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	amount, err := ParseAmount(req.Amount)
	if err != nil {
		return nil, err
	}

	if amount.IsZero() {
		return nil, types.ErrZeroAmount
	}

	ordersCaches := k.NewOrdersCaches(ctx)
	tradeCtx := types.TradeContext{
		TradeAmount:         amount,
		Context:             ctx,
		TradeDenomGiving:    req.DenomGiving,
		TradeDenomReceiving: req.DenomReceiving,
		DiscountAddress:     req.Address,
		OrdersCaches:        ordersCaches,
	}

	tradeResult, err := simulateFunc(tradeCtx, amount, req.DenomGiving, req.DenomReceiving)
	if err != nil {
		return nil, fmt.Errorf("simulate trade: %w", err)
	}

	priceGivingUSD, err := k.DenomKeeper.GetPriceInUSD(ctx, req.DenomGiving)
	if err != nil {
		return nil, fmt.Errorf("get price in USD: %w", err)
	}

	priceReceivingUSD, err := k.DenomKeeper.GetPriceInUSD(ctx, req.DenomReceiving)
	if err != nil {
		return nil, fmt.Errorf("get price in USD: %w", err)
	}

	var price string
	if tradeResult.AmountReceived.IsPositive() {
		price = tradeResult.AmountGiven.ToLegacyDec().Quo(tradeResult.AmountReceived.ToLegacyDec()).String() // C
	}

	amountGivenUSD := tradeResult.AmountGiven.ToLegacyDec().Quo(priceGivingUSD)
	amountReceivedUSD := tradeResult.AmountReceived.ToLegacyDec().Quo(priceReceivingUSD)
	spread := math.LegacyOneDec().Sub(amountReceivedUSD.Quo(amountGivenUSD))

	return &types.QuerySimulateTradeResponse{
		AmountGiven:         tradeResult.AmountGiven.String(),
		AmountGivenInUsd:    amountGivenUSD.RoundInt().String(),
		AmountReceived:      tradeResult.AmountReceived.String(),
		AmountReceivedInUsd: amountReceivedUSD.RoundInt().String(),
		Fee:                 tradeResult.FeeGiven.String(),
		Price:               price,
		PriceGivenInUsd:     priceGivingUSD.String(),
		PriceReceivedInUsd:  priceReceivingUSD.String(),
		Spread:              spread.String(),
	}, nil
}
