package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ValueKCoins(goCtx context.Context, req *types.QueryValueKCoinsRequest) (*types.QueryValueKCoinsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	sum := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.KCoins(ctx) {
		coin := k.BankKeeper.GetSupply(ctx, denom)
		price, _ := k.GetPriceInUSD(ctx, denom)
		sum = sum.Add(price.Mul(math.LegacyNewDecFromInt(coin.Amount)))
	}

	return &types.QueryValueKCoinsResponse{Value: sum.String()}, nil
}
