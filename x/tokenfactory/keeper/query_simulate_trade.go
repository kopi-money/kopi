package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/constants"

	"cosmossdk.io/math"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
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

	amountToSell, err := dexkeeper.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	var feeData FeeData
	if req.DenomGiving == constants.KUSD {
		feeData = k.calculateFees(ctx, pool, amountToSell, req.DenomGiving)
		amountToSell = amountToSell.Sub(feeData.Fee())
	}

	amountToReceive := constantProductSell(pool, req.DenomGiving, amountToSell)

	if req.DenomReceiving == constants.KUSD {
		feeData = k.calculateFees(ctx, pool, amountToReceive, req.DenomGiving)
		amountToReceive = amountToReceive.Sub(feeData.Fee())
	}

	var price math.LegacyDec
	if req.DenomReceiving == constants.KUSD {
		price = amountToSell.ToLegacyDec().Quo(amountToReceive.ToLegacyDec())
	} else {
		price = amountToReceive.ToLegacyDec().Quo(amountToSell.ToLegacyDec())
	}

	return &types.QuerySimulateTradeResponse{
		AmountGiven:    amountToSell.String(),
		AmountReceived: amountToReceive.String(),
		Fee:            feeData.Fee().String(),
		Price:          price.String(),
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

	amountToReceive, err := dexkeeper.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	var feeData FeeData
	if req.DenomGiving == constants.KUSD {
		feeData = k.calculateFees(ctx, pool, amountToReceive, req.DenomGiving)
		amountToReceive = amountToReceive.Add(feeData.Fee())
	}

	amountToGive, err := constantProductBuy(pool, req.DenomGiving, amountToReceive)
	if err != nil {
		return nil, err
	}

	if req.DenomReceiving == constants.KUSD {
		feeData = k.calculateFees(ctx, pool, amountToGive, req.DenomGiving)
		amountToGive = amountToGive.Add(feeData.Fee())
	}

	var price math.LegacyDec
	if req.DenomReceiving == constants.KUSD {
		price = amountToGive.ToLegacyDec().Quo(amountToReceive.ToLegacyDec())
	} else {
		price = amountToReceive.ToLegacyDec().Quo(amountToGive.ToLegacyDec())
	}

	return &types.QuerySimulateTradeResponse{
		AmountGiven:    amountToGive.String(),
		AmountReceived: amountToReceive.String(),
		Fee:            feeData.Fee().String(),
		Price:          price.String(),
	}, nil
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

	return types.FactoryDenom{}, types.ErrDenomDoesNotExists
}
