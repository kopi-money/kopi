package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

// CalculateParity calculates the parity of a given kCoin by checking the currency's ratio compared with the
// highest valued reference currency. The currency is above parity when the kCoin pair's ratio is lower than the
// reference pair's ratio.
func (k Keeper) CalculateParity(ctx context.Context, kCoin string) (*math.LegacyDec, string, error) {
	_, referenceDenom, err := k.GetHighestPriceDenom(ctx, kCoin)
	if err != nil {
		if errors.IsOf(err, types.ErrNoLiquidity, types.ErrNilRatio) {
			return nil, referenceDenom, nil
		}

		return nil, referenceDenom, errors.Wrap(err, "could not get highest price denom")
	}

	referenceRatio, err := k.GetRatio(ctx, referenceDenom)
	if err != nil {
		return nil, referenceDenom, err
	}

	kCoinRatio, err := k.GetRatio(ctx, kCoin)
	if err != nil {
		return nil, referenceDenom, err
	}

	parity := referenceRatio.Ratio.Quo(kCoinRatio.Ratio)
	return &parity, referenceDenom, nil
}

// GetHighestPriceDenom returns the highest valued of all reference denoms given one unit of a kCoin. For example,
// kUSD is connected to axlUSDC, axlUSDT and others. The price of those currencies can fluctuate or even depeg, so the
// most valued price is used as "true" price.
func (k Keeper) GetHighestPriceDenom(ctx context.Context, kCoin string) (math.LegacyDec, string, error) {
	var (
		referencePrice math.LegacyDec
		referenceDenom string
	)

	for _, reference := range k.DenomKeeper.ReferenceDenoms(ctx, kCoin) {
		price, err := k.CalculatePrice(ctx, kCoin, reference)
		if err != nil {
			return referencePrice, referenceDenom, errors.Wrap(err, "could not calculate price")
		}

		if referencePrice.IsNil() || price.GT(referencePrice) {
			referencePrice = price
			referenceDenom = reference
		}
	}

	return referencePrice, referenceDenom, nil
}
