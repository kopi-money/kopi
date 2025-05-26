package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k Keeper) FactoryPoolDenoms(ctx context.Context, _ *types.QueryFactoryPoolDenomsRequest) (*types.QueryFactoryPoolDenomsResponse, error) {
	var factoryPoolDenoms []types.FactoryPoolDenomResponse
	for _, factoryPoolDenom := range k.GetParams(ctx).FactoryPoolDenoms {
		factoryPoolDenoms = append(factoryPoolDenoms, types.FactoryPoolDenomResponse{
			Denom:           factoryPoolDenom.Denom,
			MinimumPoolSize: factoryPoolDenom.MinimumPoolSize.String(),
			MoveThreshold:   factoryPoolDenom.MoveThreshold.String(),
		})
	}

	return &types.QueryFactoryPoolDenomsResponse{
		FactoryPoolDenoms: factoryPoolDenoms,
	}, nil
}
