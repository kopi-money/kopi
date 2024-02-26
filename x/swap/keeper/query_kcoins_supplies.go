package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/swap/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) KCoinsSupplies(ctx context.Context, req *types.QueryKCoinsSuppliesRequest) (*types.QueryKCoinsSuppliesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	supplies := []*types.Supply{}

	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		coin := k.BankKeeper.GetSupply(ctx, kCoin)

		price, denom, err := k.DexKeeper.CalculateParity(ctx, coin.Denom)
		if err != nil {
			return nil, err
		}

		supplies = append(supplies, &types.Supply{
			Denom:          coin.Denom,
			Amount:         coin.Amount.String(),
			Price:          price.String(),
			ReferenceDenom: denom,
		})
	}

	return &types.QueryKCoinsSuppliesResponse{Supplies: supplies}, nil
}
