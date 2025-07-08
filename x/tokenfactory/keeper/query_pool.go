package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/trading"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryPool(ctx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	var (
		userKCoinAmount   string
		userFactoryAmount string
	)

	if _, err := sdk.AccAddressFromBech32(req.Address); err == nil {
		amountKCoin, amountFactory, _ := k.getLiquidity(ctx, factoryDenom.FullName, req.Address)
		userKCoinAmount = amountKCoin.String()
		userFactoryAmount = amountFactory.String()
	}

	normalizedKCoin := adjustToNormal(pool.KCoinAmount.ToLegacyDec(), 6)
	normalizedFactory := adjustToNormal(pool.FactoryDenomAmount.ToLegacyDec(), factoryDenom.Exponent)
	if !normalizedFactory.IsPositive() {
		return nil, fmt.Errorf("normalized factory is negative")
	}

	price := normalizedKCoin.Quo(normalizedFactory) // C
	supply := k.BankKeeper.GetSupply(ctx, req.FullName).Amount
	marketCap := supply.ToLegacyDec().Mul(price)

	marketCapUSD, err := k.DenomKeeper.GetValueInUSD(ctx, pool.KCoin, marketCap)
	if err != nil {
		return nil, fmt.Errorf("get market cap in usd: %w", err)
	}

	return &types.QueryPoolResponse{
		KcoinDenom:        pool.KCoin,
		KcoinAmount:       pool.KCoinAmount.String(),
		FactoryAmount:     pool.FactoryDenomAmount.String(),
		UserKcoinAmount:   userKCoinAmount,
		UserFactoryAmount: userFactoryAmount,
		Price:             price.String(),
		Marketcap:         marketCap.String(),
		MarketcapUsd:      marketCapUSD.String(),
		CreatedAt:         pool.CreatedAt,
		TradeFee:          pool.PoolFee.String(),
		UnlockPeriod:      pool.UnlockInSeconds,
	}, nil
}

func adjustToNormal(amount math.LegacyDec, exponent uint64) math.LegacyDec {
	return amount.Quo(math.LegacyNewDec(10).Power(exponent)) // C
}

func (k Keeper) QueryPoolLiquidityAddress(ctx context.Context, req *types.QueryPoolLiquidityAddressRequest) (*types.QueryPoolLiquidityAddressResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get highest usd reference: %w", err)
	}

	var (
		response           types.QueryPoolLiquidityAddressResponse
		amountKCoin        math.Int
		amountFactoryToken math.Int
		liquidityValue     math.LegacyDec
	)

	iterator := k.liquidityPools.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		pool := keyValue.Value().Value()

		amountKCoin, amountFactoryToken, err = k.getLiquidity(ctx, keyValue.Key(), req.Address)
		if err != nil {
			return nil, fmt.Errorf("get liquidity for address: %w", err)
		}

		liquidityValue, err = k.DenomKeeper.GetValueIn(ctx, pool.KCoin, referenceDenom, amountKCoin.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("kcoin amount in usd: %w", err)
		}

		response.Pools = append(response.Pools, types.PoolLiquidityAddress{
			FactoryDenomHash:   keyValue.Key(),
			AmountKcoin:        amountKCoin.String(),
			AmountFactoryToken: amountFactoryToken.String(),
			LiquidityValue:     liquidityValue.Mul(math.LegacyNewDec(2)).String(),
		})
	}

	return &response, nil
}

func (k Keeper) QueryPoolLiquidityAddressByDenom(ctx context.Context, req *types.QueryPoolLiquidityAddressRequestByDenom) (*types.PoolLiquidityAddress, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get highest usd reference: %w", err)
	}

	pool, has := k.liquidityPools.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	amountKCoin, amountFactoryToken, err := k.getLiquidity(ctx, req.FullName, req.Address)
	if err != nil {
		return nil, fmt.Errorf("get liquidity for address: %w", err)
	}

	liquidityValue, err := k.DenomKeeper.GetValueIn(ctx, pool.KCoin, referenceDenom, amountKCoin.ToLegacyDec())
	if err != nil {
		return nil, fmt.Errorf("kcoin amount in usd: %w", err)
	}

	return &types.PoolLiquidityAddress{
		FactoryDenomHash:   req.FullName,
		AmountKcoin:        amountKCoin.String(),
		AmountFactoryToken: amountFactoryToken.String(),
		LiquidityValue:     liquidityValue.String(),
	}, nil
}

func (k Keeper) QuerySimulateAddingLiquidityKCoin(ctx context.Context, req *types.QuerySimulateAddingLiquidityRequest) (*types.QuerySimulateAddingLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.Token)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	amount, err := trading.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	ratio, err := pool.GetPoolRatio()
	if err != nil {
		return nil, fmt.Errorf("pool ratio: %w", err)
	}

	amountFactory := amount.ToLegacyDec().Quo(ratio)
	return &types.QuerySimulateAddingLiquidityResponse{
		AmountKcoin:        amount.String(),
		AmountFactoryToken: amountFactory.Ceil().TruncateInt().String(),
	}, nil
}

func (k Keeper) QuerySimulateAddingLiquidityFactoryToken(ctx context.Context, req *types.QuerySimulateAddingLiquidityRequest) (*types.QuerySimulateAddingLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.Token)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	amount, err := trading.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	ratio, err := pool.GetPoolRatio()
	if err != nil {
		return nil, fmt.Errorf("pool ratio: %w", err)
	}

	amountKCoin := amount.ToLegacyDec().Mul(ratio)
	return &types.QuerySimulateAddingLiquidityResponse{
		AmountKcoin:        amountKCoin.Ceil().TruncateInt().String(),
		AmountFactoryToken: amount.String(),
	}, nil
}

func (k Keeper) QueryUSDValue(ctx context.Context, req *types.QueryUSDValueRequest) (*types.QueryUSDValueResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	amount, err := trading.ParseAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	kCoinValue, err := pool.ConvertToKCoin(amount)
	if err != nil {
		return nil, fmt.Errorf("convert to kcoin: %w", err)
	}

	usdValue, err := k.DenomKeeper.GetValueInUSD(ctx, pool.KCoin, kCoinValue)
	if err != nil {
		return nil, fmt.Errorf("get usd value: %w", err)
	}

	return &types.QueryUSDValueResponse{
		ValueUsd: usdValue.String(),
	}, nil
}
