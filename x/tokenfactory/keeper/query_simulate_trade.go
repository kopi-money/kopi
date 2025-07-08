package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/trading"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) QuerySimulateSell(ctx context.Context, req *types.QuerySimulateTradeRequest) (*types.QuerySimulateTradeResponse, error) {
	if req.DenomGiving == req.DenomReceiving {
		return nil, types.ErrSameDenom
	}

	factoryDenom, err := k.getFactoryDenom(ctx, req.DenomGiving, req.DenomReceiving)
	if err != nil {
		return nil, err
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	amountGiving, err := trading.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	tradeContext := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.SellCallbacks(),
		TradeAmount:    amountGiving,
		Pool:           pool,
		DenomGiving:    req.DenomGiving,
		DenomReceiving: req.DenomReceiving,
	}

	tradeResult, err := trading.Trade(tradeContext.ToTradeData())
	if err != nil {
		return nil, err
	}

	price, _ := tradeResult.PricePaidExact()
	priceKCoin := getPriceKCoin(price, req.DenomReceiving == pool.KCoin)
	feeAmount := trading.GetSellFee(tradeResult)

	amountGivenUSD, err := k.toUSD(ctx, pool, tradeResult.AmountGiven().ToLegacyDec(), req.DenomGiving)
	if err != nil {
		return nil, fmt.Errorf("given to usd: %w", err)
	}

	amountReceivedUSD, err := k.toUSD(ctx, pool, tradeResult.AmountReceived().ToLegacyDec(), req.DenomReceiving)
	if err != nil {
		return nil, fmt.Errorf("given to usd: %w", err)
	}

	return &types.QuerySimulateTradeResponse{
		AmountGiven:       tradeResult.AmountGiven().String(),
		AmountGivenUsd:    amountGivenUSD.String(),
		AmountReceived:    tradeResult.AmountReceived().String(),
		AmountReceivedUsd: amountReceivedUSD.String(),
		Fee:               feeAmount.String(),
		Price:             price.String(),
		PriceKcoin:        priceKCoin.String(),
	}, nil
}

func (k Keeper) QuerySimulateBuy(ctx context.Context, req *types.QuerySimulateTradeRequest) (*types.QuerySimulateTradeResponse, error) {
	if req.DenomGiving == req.DenomReceiving {
		return nil, types.ErrSameDenom
	}

	factoryDenom, err := k.getFactoryDenom(ctx, req.DenomGiving, req.DenomReceiving)
	if err != nil {
		return nil, err
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	amountToReceive, err := trading.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	tradeContext := types.TradeContext{
		Context:        ctx,
		Callbacks:      trading.BuyCallbacks(),
		TradeAmount:    amountToReceive,
		Pool:           pool,
		DenomGiving:    req.DenomGiving,
		DenomReceiving: req.DenomReceiving,
	}

	tradeResult, err := trading.Trade(tradeContext.ToTradeData())
	if err != nil {
		return nil, err
	}

	price, err := tradeResult.PricePaidExact()
	if err != nil {
		return nil, fmt.Errorf("price paid exact: %w", err)
	}

	priceKCoin := getPriceKCoin(price, req.DenomReceiving == pool.KCoin)
	feeAmount := trading.GetSellFee(tradeResult)

	amountGivenUSD, err := k.toUSD(ctx, pool, tradeResult.AmountGiven().ToLegacyDec(), req.DenomGiving)
	if err != nil {
		return nil, fmt.Errorf("given to usd: %w", err)
	}

	amountReceivedUSD, err := k.toUSD(ctx, pool, tradeResult.AmountReceived().ToLegacyDec(), req.DenomReceiving)
	if err != nil {
		return nil, fmt.Errorf("given to usd: %w", err)
	}

	return &types.QuerySimulateTradeResponse{
		AmountGiven:       tradeResult.AmountGiven().String(),
		AmountGivenUsd:    amountGivenUSD.String(),
		AmountReceived:    tradeResult.AmountReceived().String(),
		AmountReceivedUsd: amountReceivedUSD.String(),
		Fee:               feeAmount.String(),
		Price:             price.String(),
		PriceKcoin:        priceKCoin.String(),
	}, nil
}

func (k Keeper) toUSD(ctx context.Context, pool types.LiquidityPool, amount math.LegacyDec, denom string) (math.LegacyDec, error) {
	if denom != pool.KCoin {
		poolPrice, err := pool.Price()
		if err != nil {
			return math.LegacyZeroDec(), fmt.Errorf("pool price: %w", err)
		}

		amount = amount.Quo(poolPrice)
	}

	return k.DenomKeeper.GetValueInUSD(ctx, pool.KCoin, amount)
}

func (k Keeper) getFactoryDenom(ctx context.Context, denomGiving, denomReceiving string) (types.FactoryDenom, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, denomGiving)
	if has {
		return factoryDenom, nil
	}

	factoryDenom, has = k.GetDenomByFullName(ctx, denomReceiving)
	if has {
		return factoryDenom, nil
	}

	return types.FactoryDenom{}, types.ErrDenomDoesNotExist
}

func getPriceKCoin(price math.LegacyDec, boughtKCoin bool) math.LegacyDec {
	if !boughtKCoin {
		price = math.LegacyOneDec().Quo(price)
	}

	return price
}
