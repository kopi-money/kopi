package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetPool(ctx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
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
	price := normalizedKCoin.Quo(normalizedFactory)

	return &types.QueryPoolResponse{
		KcoinDenom:        pool.KCoin,
		KcoinAmount:       pool.KCoinAmount.String(),
		FactoryAmount:     pool.FactoryDenomAmount.String(),
		UserKcoinAmount:   userKCoinAmount,
		UserFactoryAmount: userFactoryAmount,
		Price:             price.String(),
		CreatedAt:         pool.CreatedAt,
	}, nil
}

func adjustToNormal(amount math.LegacyDec, exponent uint64) math.LegacyDec {
	return amount.Quo(math.LegacyNewDec(10).Power(exponent))
}
