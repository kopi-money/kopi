package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryOffers(ctx context.Context, req *types.QueryOffersRequest) (*types.QueryOffersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var offers []types.Offer
	iterator := k.offers.Iterator(ctx, nil)
	for iterator.Valid() {
		offer := iterator.GetNext()

		if req.FullName != "" && offer.FactoryDenom != req.FullName {
			continue
		}

		offers = append(offers, offer)
	}

	return &types.QueryOffersResponse{Offers: offers}, nil
}
