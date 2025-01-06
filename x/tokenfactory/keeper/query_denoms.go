package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Denoms(ctx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenomData, pageRes, err := query.CollectionPaginate(
		ctx,
		k.factoryDenoms,
		req.Pagination,
		func(key string, value types.FactoryDenom) (*types.FactoryDenomData, error) {
			supply := k.BankKeeper.GetSupply(ctx, value.FullName)
			_, hasPool := k.liquidityPools.Get(ctx, value.FullName)

			return &types.FactoryDenomData{
				Admin:       value.Admin,
				DisplayName: value.DisplayName,
				FullName:    value.FullName,
				Description: value.Description,
				IconHash:    value.IconHash,
				Symbol:      value.Symbol,
				Exponent:    value.Exponent,
				Supply:      supply.Amount.Int64(),
				HasPool:     hasPool,
				Mintable:    value.Mintable,
			}, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not get factory denoms from pagination: %w", err)
	}

	return &types.QueryDenomsResponse{
		Denoms:     factoryDenomData,
		Pagination: pageRes,
	}, nil
}
