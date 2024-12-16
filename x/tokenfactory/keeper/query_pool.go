package keeper

import (
	"context"
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

	return &types.QueryPoolResponse{
		KcoinDenom:    pool.KCoin,
		KcoinAmount:   pool.KCoinAmount.String(),
		FactoryAmount: pool.FactoryDenomAmount.String(),
	}, nil
}
