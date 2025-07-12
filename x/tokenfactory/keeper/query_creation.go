package keeper

import (
	"context"
	"strings"

	"github.com/kopi-money/kopi/x/tokenfactory/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryCreationNameExists(ctx context.Context, req *types.QueryCreationNameExistsQuery) (*types.QueryCreationExistsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var response types.QueryCreationExistsResponse
	name := strings.ToLower(req.Name)

	if err := types.IsValidDisplayName(name); err != nil {
		response.Exists = true
	}

	if !response.Exists {
		iterator := k.factoryDenoms.Iterator(ctx, nil)
		for iterator.Valid() {
			factoryDenom := iterator.GetNext()
			if strings.ToLower(factoryDenom.DisplayName) == name {
				response.Exists = true
				break
			}
		}
	}

	return &response, nil
}

func (k Keeper) QueryCreationSymbolExists(ctx context.Context, req *types.QueryCreationSymbolExistsQuery) (*types.QueryCreationExistsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var response types.QueryCreationExistsResponse
	symbol := strings.ToLower(req.Symbol)

	if err := types.IsValidSymbol(symbol); err != nil {
		response.Exists = true
	}

	if !response.Exists {
		iterator := k.factoryDenoms.Iterator(ctx, nil)
		for iterator.Valid() {
			factoryDenom := iterator.GetNext()
			if strings.ToLower(factoryDenom.Symbol) == symbol {
				response.Exists = true
				break
			}
		}
	}

	return &response, nil
}
