package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k Keeper) PricesUSD(ctx context.Context, _ *types.QueryGetPricesUSDRequest) (*types.QueryGetPricesUSDResponse, error) {
	reference, err := k.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get highest USD reference: %w", err)
	}

	var (
		prices   []types.Price
		priceUSD math.LegacyDec
	)

	for _, denom := range k.Denoms(ctx) {
		priceUSD, err = k.CalculatePrice(ctx, denom, reference)
		if err != nil {
			priceUSD = math.LegacyZeroDec()
			k.Logger().Error(fmt.Sprintf("unable to calculate price (%v): %v", denom, err))
		}

		if priceUSD.IsPositive() {
			priceUSD = math.LegacyOneDec().Quo(priceUSD)
		}

		prices = append(prices, types.Price{
			Denom: denom,
			Price: priceUSD.String(),
		})
	}

	return &types.QueryGetPricesUSDResponse{
		Prices: prices,
	}, nil
}
