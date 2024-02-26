package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) CalculatePrice(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, error) {
	price := math.LegacyOneDec()

	if denomFrom != utils.BaseCurrency {
		ratio, found := k.GetRatio(ctx, denomFrom)
		if !found || ratio.Ratio == nil {
			return math.LegacyDec{}, types.ErrNilRatio
		}

		price = price.Quo(*ratio.Ratio)
	}

	if denomTo != utils.BaseCurrency {
		ratio, found := k.GetRatio(ctx, denomTo)
		if !found || ratio.Ratio == nil {
			return math.LegacyDec{}, types.ErrNilRatio
		}

		price = price.Mul(*ratio.Ratio)
	}

	return price, nil
}

func (k Keeper) GetPriceInUSD(ctx context.Context, denom string) (math.LegacyDec, error) {
	price := math.LegacyZeroDec()
	for _, usd := range k.DenomKeeper.ReferenceDenoms(ctx, "ukusd") {
		p, err := k.CalculatePrice(ctx, denom, usd)
		if err != nil {
			return price, errors.Wrap(err, "could not calculate price")
		}

		if price.Equal(math.LegacyZeroDec()) || p.LT(price) {
			price = p
		}
	}

	return price, nil
}

func (k Keeper) GetValueInBase(ctx context.Context, denom string, amount math.Int) (math.LegacyDec, error) {
	return k.GetValueIn(ctx, denom, utils.BaseCurrency, amount)
}

func (k Keeper) GetValueInUSD(ctx context.Context, denom string, amount math.Int) (math.LegacyDec, error) {
	if amount.Equal(math.ZeroInt()) {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.GetPriceInUSD(ctx, denom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return price.Mul(amount.ToLegacyDec()), nil
}

func (k Keeper) GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.Int) (math.LegacyDec, error) {
	if amount.Equal(math.ZeroInt()) {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.CalculatePrice(ctx, denomFrom, denomTo)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return price.Mul(amount.ToLegacyDec()), nil
}
