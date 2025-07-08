package keeper

import (
	"context"
	"strconv"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryDenoms(ctx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var (
		iterator = k.factoryDenoms.Iterator(ctx, nil)
		denoms   []types.FactoryDenomData
	)

	for iterator.Valid() {
		denom := iterator.GetNext()

		supply := k.BankKeeper.GetSupply(ctx, denom.FullName)
		_, hasPool := k.liquidityPools.Get(ctx, denom.FullName)

		denoms = append(denoms, types.FactoryDenomData{
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
		})
	}

	return &types.QueryDenomsResponse{
		Denoms: denoms,
	}, nil
}
