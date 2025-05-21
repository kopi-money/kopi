package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) ValueKCoins(ctx context.Context, _ *types.QueryValueKCoinsRequest) (*types.QueryValueKCoinsResponse, error) {
	sum := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.KCoins(ctx) {
		coin := k.BankKeeper.GetSupply(ctx, denom)
		value, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, coin.Amount.ToLegacyDec())
		sum = sum.Add(value)
	}

	return &types.QueryValueKCoinsResponse{
		Value: sum.String(),
	}, nil
}
