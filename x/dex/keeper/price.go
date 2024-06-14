package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

// CalculatePrice returns the price of a given currency pair. The price is expressed how much "FROM" you need to give
// get one unit of "TO". I.e., the lower the returned value, the more valuable "FROM" is (or the less valuable "TO" is).
func (k Keeper) CalculatePrice(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, error) {
	price := math.LegacyOneDec()

	if denomFrom != utils.BaseCurrency {
		ratio, err := k.GetRatio(ctx, denomFrom)
		if err != nil {
			return price, err
		}

		price = price.Quo(ratio.Ratio)
	}

	if denomTo != utils.BaseCurrency {
		ratio, err := k.GetRatio(ctx, denomTo)
		if err != nil {
			return price, err
		}

		price = price.Mul(ratio.Ratio)
	}

	if price.Equal(math.LegacyZeroDec()) {
		return math.LegacyDec{}, types.ErrZeroPrice
	}

	price = math.LegacyOneDec().Quo(price)
	return price, nil
}

func (k Keeper) GetPriceInUSD(ctx context.Context, denom string) (math.LegacyDec, error) {
	price := math.LegacyZeroDec()
	for _, usd := range k.DenomKeeper.ReferenceDenoms(ctx, "ukusd") {
		p, err := k.CalculatePrice(ctx, denom, usd)
		if err != nil {
			return price, errors.Wrap(err, "could not calculate price")
		}

		if price.Equal(math.LegacyZeroDec()) || p.GT(price) {
			price = p
		}
	}

	return price, nil
}

func (k Keeper) GetValueInBase(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error) {
	return k.GetValueIn(ctx, denom, utils.BaseCurrency, amount)
}

func (k Keeper) GetValueInUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error) {
	if amount.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.GetPriceInUSD(ctx, denom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return amount.Quo(price), nil
}

func (k Keeper) GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error) {
	if amount.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.CalculatePrice(ctx, denomFrom, denomTo)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return amount.Quo(price), nil
}
