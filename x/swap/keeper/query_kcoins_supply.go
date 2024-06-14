package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/swap/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) KCoinSupply(ctx context.Context, req *types.QueryKCoinSupplyRequest) (*types.QueryKCoinSupplyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if !k.DenomKeeper.IsKCoin(ctx, req.Denom) {
		return nil, types.ErrNoKCoin
	}

	coin := k.BankKeeper.GetSupply(ctx, req.Denom)

	price, denom, err := k.DexKeeper.CalculateParity(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	ratioKCoin, _ := k.DexKeeper.GetRatio(ctx, req.Denom)
	ratioReference, _ := k.DexKeeper.GetRatio(ctx, denom)

	return &types.QueryKCoinSupplyResponse{
		Amount:         coin.Amount.String(),
		Price:          price.String(),
		ReferenceDenom: denom,
		RatioReference: ratioReference.Ratio.String(),
		RatioVirtual:   ratioKCoin.Ratio.String(),
	}, nil
}
