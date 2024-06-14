package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SimulateTrade(goCtx context.Context, req *types.QuerySimulateTradeRequest) (*types.QuerySimulateTradeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	amount, err := parseAmount(req.Amount)
	if err != nil {
		return nil, err
	}

	if amount.Equal(math.ZeroInt()) {
		return nil, types.ErrZeroAmount
	}

	tradeCtx := types.TradeContext{
		Context:         ctx,
		GivenAmount:     amount,
		TradeDenomStart: req.DenomFrom,
		TradeDenomEnd:   req.DenomTo,
		DiscountAddress: req.Address,
	}

	amountReceived, fee, price, err := k.TradeSimulation(tradeCtx)
	if err != nil {
		return nil, errors.Wrap(err, "could not simulate trade")
	}

	priceFromUSD, err := k.GetPriceInUSD(ctx, req.DenomFrom)
	if err != nil {
		return nil, errors.Wrap(err, "could not get price in USD")
	}

	priceToUSD, err := k.GetPriceInUSD(ctx, req.DenomTo)
	if err != nil {
		return nil, errors.Wrap(err, "could not get price in USD")
	}

	res := types.QuerySimulateTradeResponse{
		AmountReceived:      amountReceived.Int64(),
		AmountReceivedInUsd: amountReceived.ToLegacyDec().Quo(priceToUSD).RoundInt64(),
		AmountGivenInUsd:    amount.ToLegacyDec().Quo(priceFromUSD).RoundInt64(),
		Fee:                 fee.RoundInt64(),
		Price:               price.String(),
		PriceFromToUsd:      priceFromUSD.String(),
		PriceToToUsd:        priceToUSD.String(),
	}

	return &res, nil
}
