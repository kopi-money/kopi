package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/dex/types"
)

// CalculatePrice returns the price of a given currency pair. The price is expressed how much "FROM" you need to give
// get one unit of "TO". I.e., the lower the returned value, the more valuable "FROM" is (or the less valuable "TO" is).
func (k Keeper) CalculatePrice(ctx context.Context, denomGiving, denomReceiving string) (math.LegacyDec, error) {
	price := math.LegacyOneDec()

	if denomGiving != constants.BaseCurrency {
		ratio, err := k.GetRatio(ctx, denomGiving)
		if err != nil {
			return price, err
		}

		if !ratio.Ratio.IsPositive() {
			return price, fmt.Errorf("ratio is not positive")
		}

		price = price.Quo(ratio.Ratio) // C
	}

	if denomReceiving != constants.BaseCurrency {
		ratio, err := k.GetRatio(ctx, denomReceiving)
		if err != nil {
			return price, err
		}

		price = price.Mul(ratio.Ratio)
	}

	if price.IsZero() {
		return math.LegacyDec{}, types.ErrZeroPrice
	}

	price = math.LegacyOneDec().Quo(price) // C
	return price, nil
}

func (k Keeper) GetPriceInUSD(ctx context.Context, denom string) (math.LegacyDec, error) {
	referenceDenom, err := k.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not get highest usd reference: %w", err)
	}

	return k.CalculatePrice(ctx, denom, referenceDenom)
}

func (k Keeper) GetValueInFromUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error) {
	referenceDenom, err := k.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not get highest usd reference: %w", err)
	}

	return k.GetValueIn(ctx, referenceDenom, denom, amount)
}

func (k Keeper) GetHighestUSDReference(ctx context.Context) (string, error) {
	var (
		ratio math.LegacyDec
		denom string
	)

	for _, usd := range k.ReferenceDenoms(ctx, constants.KUSD) {
		r, err := k.GetRatio(ctx, usd)
		if err != nil {
			return "", err
		}

		if ratio.IsNil() || ratio.GT(r.Ratio) {
			ratio = r.Ratio
			denom = usd
		}
	}

	return denom, nil
}

func (k Keeper) GetValueInBase(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error) {
	return k.GetValueIn(ctx, denom, constants.BaseCurrency, amount)
}

func (k Keeper) GetValueInUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error) {
	if amount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.GetPriceInUSD(ctx, denom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	if !price.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("price is not positive")
	}

	value := amount.Quo(price) // C
	return value, nil
}

func (k Keeper) GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error) {
	if amount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.CalculatePrice(ctx, denomFrom, denomTo)
	if err != nil {
		return math.LegacyDec{}, err
	}

	if !price.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("price is not positive")
	}

	return amount.Quo(price), nil // C
}
