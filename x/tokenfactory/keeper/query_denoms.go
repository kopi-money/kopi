package keeper

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryDenoms(ctx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenomData, pageRes, err := query.CollectionPaginate(
		ctx,
		k.factoryDenoms,
		req.Pagination,
		func(key string, denom types.FactoryDenom) (types.FactoryDenomData, error) {
			supply := k.BankKeeper.GetSupply(ctx, denom.FullName)
			_, hasPool := k.liquidityPools.Get(ctx, denom.FullName)

			return types.FactoryDenomData{
				Admin:         denom.Admin,
				DisplayName:   denom.DisplayName,
				FullName:      denom.FullName,
				Description:   denom.Description,
				Website:       denom.Website,
				IconHash:      denom.IconHash,
				Symbol:        denom.Symbol,
				Exponent:      strconv.Itoa(int(denom.Exponent)),
				Supply:        supply.Amount.String(),
				HasPool:       hasPool,
				Mintable:      denom.Mintable,
				CategoryIndex: denom.CategoryIndex,
			}, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("get factory denoms from pagination: %w", err)
	}

	return &types.QueryDenomsResponse{
		Denoms:     factoryDenomData,
		Pagination: pageRes,
	}, nil
}
