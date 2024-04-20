package keeper

import (
	"context"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/utils"

	"cosmossdk.io/store/prefix"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) GetLiquidityPairCount(ctx context.Context) uint64 {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityPairCount))

	byteKey := types.Key(types.KeyLiquidityPairCount)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetLiquidityPairCount(ctx context.Context, count uint64) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyLiquidityPairCount))
	byteKey := types.Key(types.KeyLiquidityPairCount)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// SetLiquidityPair set a specific liquidityPair in the store
func (k Keeper) SetLiquidityPair(ctx context.Context, liquidityPair types.LiquidityPair) {
	if liquidityPair.Denom == "" {
		panic("denom must not be empty")
	}
	if liquidityPair.VirtualBase.LT(math.LegacyZeroDec()) {
		panic("liquidityPair.VirtualBase must not be less than zero")
	}
	if liquidityPair.VirtualOther.LT(math.LegacyZeroDec()) {
		panic("liquidityPair.VirtualOther must not be less than zero")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidityPair))
	b := k.cdc.MustMarshal(&liquidityPair)
	store.Set(types.KeyString(liquidityPair.Denom), b)
}

func (k Keeper) GetLiquidityPair(ctx context.Context, denom string) (val types.LiquidityPair, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidityPair))
	b := store.Get(types.KeyString(denom))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) GetAllLiquidityPair(ctx context.Context) (list []types.LiquidityPair) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidityPair))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.LiquidityPair
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) InitPairs(ctx context.Context) {
	liqBase, found := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	if found && !liqBase.IsNil() {
		for _, denom := range k.DenomKeeper.Denoms(ctx) {
			if denom != utils.BaseCurrency {
				k.initPair(ctx, liqBase, denom)
			}
		}
	}
}

func (k Keeper) initPair(ctx context.Context, liqBase math.Int, denom string) {
	fac := k.DenomKeeper.InitialVirtualLiquidityFactor(ctx, denom)

	pair, found := k.GetLiquidityPair(ctx, denom)
	if !found {
		pair.Denom = denom
		pair.VirtualBase = math.LegacyZeroDec()
		pair.VirtualOther = liqBase.ToLegacyDec().Mul(fac)
		k.SetLiquidityPair(ctx, pair)
	}

	ratio, found := k.GetRatio(ctx, denom)
	if !found || found && ratio.Ratio == nil {
		ratio.Denom = denom
		ratio.Ratio = &fac
		k.SetRatio(ctx, ratio)
	}
}

func (k Keeper) updatePairs(ctx context.Context) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != utils.BaseCurrency {
			k.updatePair(ctx, denom)
		}
	}
}

// updatePair updates the virtual liquidity for a denom pair. This function only is called after having added liqudity
// to a pair.
func (k Keeper) updatePair(ctx context.Context, denom string) {
	pair, _ := k.GetLiquidityPair(ctx, denom)
	ratio, found := k.GetRatio(ctx, denom)

	if ratio.Ratio != nil {
		liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		liqOther, _ := k.GetLiquiditySum(ctx, denom)

		liqBaseDec := liqBase.ToLegacyDec()
		liqOtherDec := liqOther.ToLegacyDec()

		if liqBaseDec.Mul(*ratio.Ratio).GT(liqOtherDec) {
			pair.VirtualBase = math.LegacyZeroDec()
			pair.VirtualOther = k.calcVirtualAmountOther(ctx, *ratio.Ratio, denom)
		} else {
			pair.VirtualBase = k.calcVirtualAmountBase(ctx, *ratio.Ratio, denom)
			pair.VirtualOther = math.LegacyZeroDec()
		}

		k.SetLiquidityPair(ctx, pair)
	} else if !found || ratio.Ratio == nil {
		liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		k.initPair(ctx, liqBase, denom)
	}
}

func (k Keeper) calcVirtualAmountOther(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liqOther, _ := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqBaseDec.Mul(ratio).Sub(liqOtherDec)
}

func (k Keeper) calcVirtualAmountBase(ctx context.Context, ratio math.LegacyDec, denom string) math.LegacyDec {
	liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liqOther, _ := k.GetLiquiditySum(ctx, denom)
	liqBaseDec := liqBase.ToLegacyDec()
	liqOtherDec := liqOther.ToLegacyDec()

	return liqOtherDec.Quo(ratio).Sub(liqBaseDec)
}

func (k Keeper) PairRatio(ctx context.Context, denom string) *math.LegacyDec {
	liqBase := k.GetFullLiquidityBase(ctx, denom)
	liqOther := k.GetFullLiquidityOther(ctx, denom)
	if !liqOther.GT(math.LegacyZeroDec()) {
		return nil
	}

	ratio := liqBase.Mul(liqOther)
	return &ratio
}

func (k Keeper) GetFullLiquidity(ctx context.Context, denom, other string) math.LegacyDec {
	if denom == utils.BaseCurrency {
		return k.GetFullLiquidityBase(ctx, other)
	} else {
		return k.GetFullLiquidityOther(ctx, denom)
	}
}

func (k Keeper) GetFullLiquidityBaseOther(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, math.LegacyDec) {
	var liq1, liq2 math.LegacyDec

	if denomFrom == utils.BaseCurrency {
		liq1 = k.GetFullLiquidityBase(ctx, denomTo)
		liq2 = k.GetFullLiquidityOther(ctx, denomTo)
	} else {
		liq1 = k.GetFullLiquidityOther(ctx, denomFrom)
		liq2 = k.GetFullLiquidityBase(ctx, denomFrom)
	}

	return liq1, liq2
}

func (k Keeper) GetFullLiquidityBase(ctx context.Context, denomOther string) math.LegacyDec {
	if denomOther == utils.BaseCurrency {
		panic("other denom cannot be base currency")
	}

	liq1, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	liq2, _ := k.GetLiquidityPair(ctx, denomOther)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualBase)
}

func (k Keeper) GetFullLiquidityOther(ctx context.Context, denom string) math.LegacyDec {
	liq1, _ := k.GetLiquiditySum(ctx, denom)
	liq2, _ := k.GetLiquidityPair(ctx, denom)
	return sumLiquidity(liq1.ToLegacyDec(), liq2.VirtualOther)
}

func sumLiquidity(actual, virtual math.LegacyDec) math.LegacyDec {
	if actual.IsNil() {
		panic("actual liquidity is nil")
	}
	if virtual.IsNil() {
		virtual = math.LegacyZeroDec()
	}

	return actual.Add(virtual)
}
