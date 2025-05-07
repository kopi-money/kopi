package keeper

import (
	"context"
	"strings"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) QueryCreationNameExists(ctx context.Context, req *types.QueryCreationNameExistsQuery) (*types.QueryCreationExistsResponse, error) {
	var response types.QueryCreationExistsResponse
	name := strings.ToLower(req.Name)

	iterator := k.factoryDenoms.Iterator(ctx, nil)
	for iterator.Valid() {
		factoryDenom := iterator.GetNext()
		if strings.ToLower(factoryDenom.DisplayName) == name {
			response.Exists = true
			break
		}
	}

	return &response, nil
}

func (k Keeper) QueryCreationSymbolExists(ctx context.Context, req *types.QueryCreationSymbolExistsQuery) (*types.QueryCreationExistsResponse, error) {
	var response types.QueryCreationExistsResponse
	symbol := strings.ToLower(req.Symbol)

	iterator := k.factoryDenoms.Iterator(ctx, nil)
	for iterator.Valid() {
		factoryDenom := iterator.GetNext()
		if strings.ToLower(factoryDenom.Symbol) == symbol {
			response.Exists = true
			break
		}
	}

	return &response, nil
}
