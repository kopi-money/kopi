package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetTradeAmount(ctx context.Context, address string) math.LegacyDec {
	tradeAmount, has := k.tradeAmounts.Get(ctx, address)
	if !has {
		return math.LegacyZeroDec()
	}

	return tradeAmount.Amount
}

func (k Keeper) AddTradeAmount(ctx context.Context, address string, amount math.Int) {
	tradeAmount, has := k.tradeAmounts.Get(ctx, address)
	if !has {
		tradeAmount = types.WalletTradeAmount{
			Address: address,
			Amount:  math.LegacyZeroDec(),
		}
	}

	tradeAmount.Amount = tradeAmount.Amount.Add(amount.ToLegacyDec())
	k.tradeAmounts.Set(ctx, address, tradeAmount)
}

func (k Keeper) TradeAmountDecay(ctx context.Context) {
	iterator := k.tradeAmounts.Iterator(ctx, nil)
	decayFactor := k.GetParams(ctx).TradeAmountDecay

	for iterator.Valid() {
		tradeAmount := iterator.GetNext()

		if tradeAmount.Amount.IsNil() || tradeAmount.Amount.LT(math.LegacyNewDec(1_000_000)) {
			k.tradeAmounts.Remove(ctx, tradeAmount.Address)
		} else {
			tradeAmount.Amount = tradeAmount.Amount.Mul(decayFactor)
			k.tradeAmounts.Set(ctx, tradeAmount.Address, tradeAmount)
		}
	}
}

func (k Keeper) getTradeDiscount(ctx context.Context, address string, excludeFromDiscount bool) math.LegacyDec {
	if excludeFromDiscount {
		return math.LegacyZeroDec()
	}

	if address == "" {
		return math.LegacyZeroDec()
	}

	tradeAmount, has := k.tradeAmounts.Get(ctx, address)
	if !has {
		return math.LegacyZeroDec()
	}

	if tradeAmount.Amount.IsZero() {
		return math.LegacyZeroDec()
	}

	discountLevels := k.GetParams(ctx).DiscountLevels
	discountAmount := math.LegacyZeroDec()
	discount := math.LegacyZeroDec()

	// Iterate over all discount levels to check which is the best
	for _, discountLevel := range discountLevels {
		if discountLevel.TradeAmount.GTE(discountAmount) && tradeAmount.Amount.GTE(discountLevel.TradeAmount) {
			discountAmount = discountLevel.TradeAmount
			discount = discountLevel.Discount
		}
	}

	return discount
}
