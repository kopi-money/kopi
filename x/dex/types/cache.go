package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/measurement"
)

type LoadAccAddress func() sdk.AccAddress
type LoadFee func() math.LegacyDec
type LoadPoolBalance func() *CoinMap
type LoadLiquidityPair func(denom string) LiquidityPair
type LoadLiquidity func(denom string) []Liquidity

func NewOrderCaches(lat, lar, lal, lao LoadAccAddress, ltf, lrfs, lof, lpf LoadFee, lpbl LoadPoolBalance, ll LoadLiquidity) *OrdersCaches {
	return &OrdersCaches{
		AccPoolTrade:     newItemCache(lat),
		AccPoolReserve:   newItemCache(lar),
		AccPoolLiquidity: newItemCache(lal),
		AccPoolOrders:    newItemCache(lao),
		TradeFee:         newItemCache(ltf),
		ReserveFeeShare:  newItemCache(lrfs),
		OrderFee:         newItemCache(lof),
		ProviderFee:      newItemCache(lpf),
		LiquidityPool:    newItemCache(lpbl),
		LiquidityMap:     newLiquidityMap(ll),

		PriceAmounts:          make(map[Pair]math.LegacyDec),
		PriceMaxAmounts:       make(map[string]math.LegacyDec),
		MaximumTradableAmount: make(map[string]*math.LegacyDec),
	}
}

type Pair struct {
	DenomFrom string
	DenomTo   string
}

type CoinMap struct {
	cm map[string]math.Int
}

func NewCoinMap(coins sdk.Coins) *CoinMap {
	coinMap := make(map[string]math.Int)
	for _, coin := range coins {
		coinMap[coin.Denom] = coin.Amount
	}

	return &CoinMap{coinMap}
}

func (cm *CoinMap) AmountOf(denom string) math.Int {
	amount, has := cm.cm[denom]
	if !has {
		return math.ZeroInt()
	}

	return amount
}

func (cm *CoinMap) Sub(denom string, subAmount math.Int) {
	amount, has := cm.cm[denom]
	if has {
		newAmount := amount.Sub(subAmount)
		if newAmount.LT(math.ZeroInt()) {
			panic(fmt.Sprintf("negative coin amount for %v", denom))
		}

		cm.cm[denom] = newAmount
		return
	}

	panic(fmt.Sprintf("cannot sub denom that does not exist (%v)", denom))
}

func (cm *CoinMap) Add(denom string, addAmount math.Int) {
	amount, has := cm.cm[denom]
	if has {
		cm.cm[denom] = amount.Add(addAmount)
	} else {
		cm.cm[denom] = addAmount
	}
}

func (cm *CoinMap) Coins() sdk.Coins {
	var coinList []sdk.Coin
	for denom, amount := range cm.cm {
		coinList = append(coinList, sdk.NewCoin(denom, amount))
	}

	return sdk.NewCoins(coinList...)
}

type OrdersCaches struct {
	AccPoolReserve        *ItemCache[sdk.AccAddress]
	AccPoolTrade          *ItemCache[sdk.AccAddress]
	AccPoolLiquidity      *ItemCache[sdk.AccAddress]
	AccPoolOrders         *ItemCache[sdk.AccAddress]
	TradeFee              *ItemCache[math.LegacyDec]
	ReserveFeeShare       *ItemCache[math.LegacyDec]
	OrderFee              *ItemCache[math.LegacyDec]
	ProviderFee           *ItemCache[math.LegacyDec]
	LiquidityPool         *ItemCache[*CoinMap]
	ReimbursementPool     *ItemCache[*CoinMap]
	LiquidityPair         *MapCache[LiquidityPair]
	AdditionalLiquidity   *MapCache[math.LegacyDec]
	PriceAmounts          map[Pair]math.LegacyDec
	PriceMaxAmounts       map[string]math.LegacyDec
	LiquidityMap          *LiquidityMap
	MaximumTradableAmount map[string]*math.LegacyDec
	Measurement           *measurement.Measurement
}

func (oc *OrdersCaches) Clear() {
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
	if ic.item != nil {
		return *ic.item
	}

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

func NewMapCache[T any](loader func(string) T) *MapCache[T] {
	return &MapCache[T]{
		loader: loader,
		m:      make(map[string]T),
	}
}

func (mc *MapCache[T]) Set(denom string, t T) {
	mc.m[denom] = t
}

func (mc *MapCache[T]) Get(denom string) T {
	value, has := mc.m[denom]
	if !has {
		value = mc.loader(denom)
		mc.m[denom] = value
	}

	return value
}

func (mc *MapCache[T]) GetHas(denom string) (T, bool) {
	value, has := mc.m[denom]
	if !has {
		value = mc.loader(denom)
		mc.m[denom] = value
	}

	return value, has
}

func (mc *MapCache[T]) Clear() {
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
