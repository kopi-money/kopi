package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type LoadPoolBalance func() sdk.Coins
type LoadLiquidityPair func(denom string) LiquidityPair
type LoadFullLiquidity func(denom string) math.LegacyDec
type LoadLiquidity func(denom string) []Liquidity

func NewOrderCaches(lpb LoadPoolBalance, llp LoadLiquidityPair, lflb, lflo LoadFullLiquidity, ll LoadLiquidity) *OrdersCaches {
	return &OrdersCaches{
		DexPool:            newItemCache(lpb),
		LiquidityPair:      newOrderCache(llp),
		FullLiquidityBase:  newOrderCache(lflb),
		FullLiquidityOther: newOrderCache(lflo),
		LiquidityMap:       newLiquidityMap(ll),

		PriceAmounts: make(map[Pair]math.LegacyDec),
	}
}

type Pair struct {
	DenomFrom string
	DenomTo   string
}

type OrdersCaches struct {
	DexPool            *ItemCache[sdk.Coins]
	LiquidityPair      *MapCache[LiquidityPair]
	FullLiquidityBase  *MapCache[math.LegacyDec]
	FullLiquidityOther *MapCache[math.LegacyDec]
	PriceAmounts       map[Pair]math.LegacyDec
	LiquidityMap       *LiquidityMap
}

func (oc *OrdersCaches) Clear() {
	oc.FullLiquidityBase.clear()
	oc.FullLiquidityOther.clear()
	oc.PriceAmounts = make(map[Pair]math.LegacyDec)
}

type ItemCache[T any] struct {
	loader func() T
	item   *T
}

func newItemCache[T any](loader func() T) *ItemCache[T] {
	return &ItemCache[T]{
		loader: loader,
	}
}

func (ic *ItemCache[T]) Set(t T) {
	ic.item = &t
}

func (ic *ItemCache[T]) Get() T {
	//if ic.item != nil {
	//	return *ic.item
	//}

	item := ic.loader()
	ic.item = &item
	return item
}

func (ic *ItemCache[T]) clear() {
	ic.item = nil
}

type MapCache[T any] struct {
	loader func(string) T
	m      map[string]T
}

func newOrderCache[T any](loader func(string) T) *MapCache[T] {
	return &MapCache[T]{
		loader: loader,
		m:      make(map[string]T),
	}
}

func (mc *MapCache[T]) Set(denom string, t T) {
	mc.m[denom] = t
}

func (mc *MapCache[T]) Get(denom string) T {
	pair, has := mc.m[denom]
	if has {
		return pair
	}

	pair = mc.loader(denom)
	mc.m[denom] = pair
	return pair
}

func (mc *MapCache[T]) clear() {
	mc.m = make(map[string]T)
}

type LiquidityMap struct {
	m      map[string][]Liquidity
	loader func(string) []Liquidity
}

func newLiquidityMap(loader func(string) []Liquidity) *LiquidityMap {
	return &LiquidityMap{
		m:      make(map[string][]Liquidity),
		loader: loader,
	}
}

func (lm *LiquidityMap) Get(denom string) LiquidityList {
	list, has := lm.m[denom]
	if has {
		return list
	}

	lm.m[denom] = lm.loader(denom)
	return lm.m[denom]
}

func (lm *LiquidityMap) Set(denom string, list []Liquidity) {
	lm.m[denom] = list
}

type LiquidityList []Liquidity

func (ll LiquidityList) DeleteByLiquidityIndexes(deleteIndexes []uint64) (list []Liquidity) {
	for _, l := range ll {
		seen := false
		for i, deleteIndex := range deleteIndexes {
			if deleteIndex == l.Index {
				seen = true
				deleteIndexes = append(deleteIndexes[:i], deleteIndexes[i+1:]...)
				break
			}
		}

		if !seen {
			list = append(list, l)
		}
	}

	return
}
