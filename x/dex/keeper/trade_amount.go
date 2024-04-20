package keeper

import (
	"context"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SetTradeAmount sets a specific order in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetTradeAmount(ctx context.Context, tradeAmount types.WalletTradeAmount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixTradeAmount))
	b := k.cdc.MustMarshal(&tradeAmount)
	store.Set(types.KeyString(tradeAmount.Address), b)
}

func (k Keeper) AddTradeAmount(ctx context.Context, address string, amount math.Int) {
	tradeAmount := k.GetTradeAmount(ctx, address)
	tradeAmount.Amount = tradeAmount.Amount.Add(amount.ToLegacyDec())
	k.SetTradeAmount(ctx, tradeAmount)
}

// GetTradeAmount returns a order from its id
func (k Keeper) GetTradeAmount(ctx context.Context, address string) types.WalletTradeAmount {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixTradeAmount))
	b := store.Get(types.KeyString(address))
	if b == nil {
		return types.WalletTradeAmount{
			Address: address,
			Amount:  math.LegacyZeroDec(),
		}
	}

	var val types.WalletTradeAmount
	k.cdc.MustUnmarshal(b, &val)
	return val
}

// RemoveTradeAmount removes a order from the store
func (k Keeper) RemoveTradeAmount(ctx context.Context, tradeAmount types.WalletTradeAmount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixTradeAmount))
	store.Delete(types.KeyString(tradeAmount.Address))
}

func (k Keeper) TradeAmountsIterator(ctx context.Context) storetypes.Iterator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixTradeAmount))
	return storetypes.KVStorePrefixIterator(store, []byte{})
}

func (k Keeper) TradeAmountDecay(ctx context.Context) {
	iterator := k.TradeAmountsIterator(ctx)
	defer iterator.Close()

	decayFactor := k.GetParams(ctx).TradeAmountDecay

	for ; iterator.Valid(); iterator.Next() {
		var tradeAmount types.WalletTradeAmount
		k.cdc.MustUnmarshal(iterator.Value(), &tradeAmount)

		tradeAmount.Amount = tradeAmount.Amount.Mul(decayFactor)

		if tradeAmount.Amount.LT(math.LegacyNewDec(1_000_000)) {
			k.RemoveTradeAmount(ctx, tradeAmount)
		} else {
			k.SetTradeAmount(ctx, tradeAmount)
		}
	}
}

func (k Keeper) getTradeDiscount(ctx context.Context, address string) math.LegacyDec {
	tradeAmount := k.GetTradeAmount(ctx, address)
	if tradeAmount.Amount.Equal(math.LegacyZeroDec()) {
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
