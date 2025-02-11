package keeper

import (
	"context"
	"fmt"

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

	amountToGiveGross, err := dexkeeper.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	var (
		feeDataReceiving = newFeeData()
		feeDataGiving    = newFeeData()
	)

	amountToGiveNet := amountToGiveGross
	if req.DenomGiving == pool.KCoin {
		feeDataGiving = k.calculateFees(ctx, pool, amountToGiveGross)
		amountToGiveNet = amountToGiveGross.Sub(feeDataGiving.Fee())
	}

	amountToReceiveGross := constantProductSell(pool, req.DenomGiving, amountToGiveNet)
	amountToReceiveNet := amountToReceiveGross
	if req.DenomReceiving == pool.KCoin {
		feeDataReceiving = k.calculateFees(ctx, pool, amountToReceiveGross)
		amountToReceiveNet = amountToReceiveGross.Sub(feeDataReceiving.Fee())
	}

	var price math.LegacyDec
	if req.DenomReceiving == pool.KCoin {
		price = amountToReceiveNet.ToLegacyDec().Quo(amountToGiveGross.ToLegacyDec()) // C
	} else {
		price = amountToGiveGross.ToLegacyDec().Quo(amountToReceiveNet.ToLegacyDec()) // C
	}

	feeData := getFeeData(feeDataGiving, feeDataReceiving)
	return &types.QuerySimulateTradeResponse{
		AmountGiven:    amountToGiveGross.String(),
		AmountReceived: amountToReceiveNet.String(),
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

	amountToReceiveNet, err := dexkeeper.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	var (
		feeDataReceiving = newFeeData()
		feeDataGiving    = newFeeData()
	)

	// If the trade is to buy a kCoin, the amount to be bought has to be larger than requested because the amount used
	// for the trade fee has to be bought as well.
	amountToReceiveGross := amountToReceiveNet
	if req.DenomReceiving == pool.KCoin {
		feeDataReceiving = k.calculateFees(ctx, pool, amountToReceiveNet)
		amountToReceiveGross = amountToReceiveNet.Add(feeDataReceiving.Fee())
	}

	amountToGiveNet, err := constantProductBuy(pool, req.DenomGiving, amountToReceiveGross)
	if err != nil {
		return nil, err
	}

	// If the trade is to buy a factory token, the amount to give has to be larger than the calculated amount as to
	// cover the trade we.
	amountToGiveGross := amountToGiveNet
	if req.DenomGiving == pool.KCoin {
		feeDataGiving = k.calculateFees(ctx, pool, amountToGiveNet)
		amountToGiveGross = amountToGiveNet.Add(feeDataGiving.Fee())
	}

	var price math.LegacyDec
	if req.DenomReceiving == pool.KCoin {
		price = amountToReceiveNet.ToLegacyDec().Quo(amountToGiveGross.ToLegacyDec()) // C
	} else {
		price = amountToGiveGross.ToLegacyDec().Quo(amountToReceiveNet.ToLegacyDec()) // C
	}

	feeData := getFeeData(feeDataGiving, feeDataReceiving)
	return &types.QuerySimulateTradeResponse{
		AmountGiven:    amountToGiveGross.String(),
		AmountReceived: amountToReceiveNet.String(),
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
