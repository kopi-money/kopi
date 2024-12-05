package cache

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CollectionNestedMap[K, V any] interface {
	Get(context.Context, K) (V, error)
	Iterate(context.Context, collections.Ranger[K]) (collections.Iterator[K, V], error)
	Set(context.Context, K, V) error
	Remove(context.Context, K) error
	GetName() string
}

type NestedMapTransaction[K1, K2 ordered, V any] struct {
	key      TXKey
	changes  *NestedOrderedList[K1, K2, Entry[V]]
	previous *NestedOrderedList[K1, K2, Entry[V]]

	removals []collections.Pair[K1, K2]
}

func (nmt NestedMapTransaction[K1, K2, V]) getRemovals(key1 K1) (list []K2) {
	for _, removal := range nmt.removals {
		if removal.K1() == key1 {
			list = append(list, removal.K2())
		}
	}

	return
}

func (nmt *NestedMapTransaction[K1, K2, V]) remove(key1 K1, key2 K2, previous Entry[V]) {
	nmt.changes.Set(key1, key2, Entry[V]{})

	if has := nmt.previous.Has(key1, key2); !has {
		nmt.previous.Set(key1, key2, previous)
	}
}

func (nmt *NestedMapTransaction[K1, K2, V]) set(key1 K1, key2 K2, newValue, previous Entry[V]) {
	nmt.changes.Set(key1, key2, newValue)

	if has := nmt.previous.Has(key1, key2); !has {
		nmt.previous.Set(key1, key2, previous)
	}
}

type NestedMapTransactions[K1, K2 ordered, V any] struct {
	sync.RWMutex

	transactions []*NestedMapTransaction[K1, K2, V]
}

func (nmt *NestedMapTransactions[K1, K2, V]) Get(key TXKey) *NestedMapTransaction[K1, K2, V] {
	nestedMapTransaction := nmt.get(key)
	if nestedMapTransaction != nil {
		return nestedMapTransaction
	}

	nestedMapTransaction = &NestedMapTransaction[K1, K2, V]{
		key:      key,
		changes:  &NestedOrderedList[K1, K2, Entry[V]]{},
		previous: &NestedOrderedList[K1, K2, Entry[V]]{},
	}
	nmt.set(nestedMapTransaction)

	return nestedMapTransaction
}

func (nmt *NestedMapTransactions[K1, K2, V]) get(key TXKey) *NestedMapTransaction[K1, K2, V] {
	nmt.RLock()
	defer nmt.RUnlock()

	for _, mapTransaction := range nmt.transactions {
		if mapTransaction.key == key {
			return mapTransaction
		}
	}

	return nil
}

func (nmt *NestedMapTransactions[K1, K2, V]) set(nestedMapTransaction *NestedMapTransaction[K1, K2, V]) {
	nmt.Lock()
	defer nmt.Unlock()

	nmt.transactions = append(nmt.transactions, nestedMapTransaction)
}

func (nmt *NestedMapTransactions[K1, K2, V]) remove(key TXKey) {
	nmt.Lock()
	defer nmt.Unlock()

	index := -1
	for i, mapTransaction := range nmt.transactions {
		if mapTransaction.key.equals(key) {
			index = i
			break
		}
	}

	if index != -1 {
		nmt.transactions = append(nmt.transactions[:index], nmt.transactions[index+1:]...)
	}
}

type NestedMapCache[K1, K2 ordered, V any] struct {
	sync.Mutex

	kc     codec.KeyCodec[collections.Pair[K1, K2]]
	vc     codec.ValueCodec[V]
	prefix []byte

	collection    collections.Map[collections.Pair[K1, K2], V]
	cache         *NestedOrderedList[K1, K2, Entry[V]]
	transactions  *NestedMapTransactions[K1, K2, V]
	initialized   bool
	currentHeight int64
}

func NewNestedMapCache[K1, K2 ordered, V any](sb *collections.SchemaBuilder, prefix []byte, name string, kc codec.KeyCodec[collections.Pair[K1, K2]], vc codec.ValueCodec[V], caches *Caches) *NestedMapCache[K1, K2, V] {
	mc := &NestedMapCache[K1, K2, V]{
		kc:     kc,
		vc:     vc,
		prefix: prefix,

		cache:        &NestedOrderedList[K1, K2, Entry[V]]{},
		transactions: &NestedMapTransactions[K1, K2, V]{},
		collection:   collections.NewMap(sb, prefix, name, kc, vc),
	}

	*caches = append(*caches, mc)
	return mc
}

func (nmc *NestedMapCache[K1, K2, V]) NumRunningTransactions() int {
	return len(nmc.transactions.transactions)
}

func (nmc *NestedMapCache[K1, K2, V]) Initialize(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	nmc.currentHeight = sdkCtx.BlockHeight()

	if nmc.initialized {
		return nil
	}

	iterator, err := nmc.collection.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not create collection iterator: %w", err)
	}

	var key collections.Pair[K1, K2]
	for ; iterator.Valid(); iterator.Next() {
		key, err = iterator.Key()
		if err != nil {
			return fmt.Errorf("could not get key: %w", err)
		}

		entry, has := nmc.loadFromStorage(ctx, key.K1(), key.K2())
		if has {
			nmc.cache.Set(key.K1(), key.K2(), entry)
		}
	}

	nmc.initialized = true

	return nil
}

func (nmc *NestedMapCache[K1, K2, V]) Has(ctx context.Context, key1 K1, key2 K2) bool {
	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := nmc.transactions.Get(*txKey)
		change, has := mapTransaction.changes.Get(key1, key2)
		if has {
			return hasEntry(ctx, change)
		}
	}

	if !useCache(ctx, nmc.currentHeight) {
		return nmc.Has(ctx, key1, key2)
	}

	entry, has := nmc.cache.Get(key1, key2)
	if has {
		return hasEntry(ctx, entry)
	}

	return false
}

func (nmc *NestedMapCache[K1, K2, V]) Get(ctx context.Context, key1 K1, key2 K2) (V, bool) {
	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := nmc.transactions.Get(*txKey)
		change, has := mapTransaction.changes.Get(key1, key2)
		if has {
			return getEntry(ctx, change)
		}
	}

	if !useCache(ctx, nmc.currentHeight) {
		entry, has := nmc.loadFromStorage(ctx, key1, key2)
		if has {
			return *entry.value, true
		} else {
			var v V
			return v, false
		}
	}

	entry, has := nmc.cache.Get(key1, key2)
	if has {
		return getEntry(ctx, entry)
	}

	var v V
	return v, false
}

func (nmc *NestedMapCache[K1, K2, V]) loadFromStorage(ctx context.Context, key1 K1, key2 K2) (Entry[V], bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	gasMeter := sdkCtx.GasMeter()
	ctx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	key := collections.Join(key1, key2)
	value, err := nmc.collection.Get(ctx, key)
	ctx = sdkCtx.WithGasMeter(gasMeter)

	if err != nil {
		return Entry[V]{}, false
	}

	return Entry[V]{
		value: &value,
		cost:  CalculateReadCostMap(nmc.prefix, nmc.kc, nmc.vc, key, value),
	}, true
}

func (nmc *NestedMapCache[K1, K2, V]) Set(ctx context.Context, key1 K1, key2 K2, value V) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		getTXKey(ctx)
		panic("calling set without initialized cache transaction")
	}

	previous, has := nmc.cache.Get(key1, key2)
	if !has {
		previous = Entry[V]{}
	}

	key := collections.Join(key1, key2)
	newEntry := Entry[V]{
		value: &value,
		cost:  CalculateReadCostMap(nmc.prefix, nmc.kc, nmc.vc, key, value),
	}

	mapTransaction := nmc.transactions.Get(*txKey)
	mapTransaction.set(key1, key2, newEntry, previous)
}

func (nmc *NestedMapCache[K1, K2, V]) Remove(ctx context.Context, key1 K1, key2 K2) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling set without initialized cache transaction")
	}

	previous, has := nmc.cache.Get(key1, key2)
	if !has {
		previous = Entry[V]{}
	}

	mapTransaction := nmc.transactions.Get(*txKey)
	mapTransaction.remove(key1, key2, previous)
}

// Iterator returns an iterator which contains a list of all keys. Since the cache doesn't know about all keys, they
// have to be loaded from storage first. Then interim changes to the data have to be applied to the keys, i.e.
// adding new ones or removes those that have been deleted. If new keys are added, the list has to be sorted once more.
func (nmc *NestedMapCache[K1, K2, V]) Iterator(ctx context.Context, rng collections.Ranger[collections.Pair[K1, K2]], key1 K1) Iterator[K2, V] {
	var changes *OrderedList[K2, Entry[V]]
	var removals []K2

	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := nmc.transactions.Get(*txKey)
		changes = mapTransaction.changes.GetInnerOrderedList(key1)
		removals = mapTransaction.getRemovals(key1)
	} else {
		changes = &OrderedList[K2, Entry[V]]{}
	}

	cache := nmc.cache.GetInnerOrderedList(key1)

	valueGetter := func(key2 K2) V {
		v, _ := nmc.Get(ctx, key1, key2)
		return v
	}

	createIterator := func() Iterator[K2, V] {
		iterator, _ := nmc.collection.Iterate(ctx, rng)
		return NestedIterator[K1, K2, V]{
			iterator: iterator,
		}
	}

	return newIterator[K2, V](ctx, cache, changes, valueGetter, removals, createIterator, nmc.currentHeight)
}

func (nmc *NestedMapCache[K1, K2, V]) Size() int {
	return nmc.cache.Size()
}

func (nmc *NestedMapCache[K1, K2, V]) CommitToDB(ctx context.Context) error {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	for _, outerChangeList := range nmc.transactions.Get(*txKey).changes.lists {
		for _, change := range outerChangeList.value {
			key := collections.Join(outerChangeList.key, change.key)
			if change.value.value != nil {
				if err := nmc.collection.Set(ctx, key, *change.value.value); err != nil {
					return fmt.Errorf("could not add value to collection: %w", err)
				}
			} else {
				if err := nmc.collection.Remove(ctx, key); err != nil {
					return fmt.Errorf("could not remove value from collection: %w", err)
				}
			}
		}
	}

	return nil
}

func (nmc *NestedMapCache[K1, K2, V]) Rollback(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	// Setting an infinite gas meter because we never want the following actions to fail due to out of gas reasons.
	if sdkCtx, ok := ctx.(sdk.Context); ok {
		sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		ctx = sdkCtx
	}

	for _, outerPreviousList := range nmc.transactions.Get(*txKey).previous.lists {
		for _, change := range outerPreviousList.value {
			key := collections.Join(outerPreviousList.key, change.key)
			if change.value.value != nil {
				_ = nmc.collection.Set(ctx, key, *change.value.value)
			} else {
				_ = nmc.collection.Remove(ctx, key)
			}
		}
	}
}

func (nmc *NestedMapCache[K1, K2, V]) CommitToCache(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	for _, outerChangeList := range nmc.transactions.Get(*txKey).changes.lists {
		for _, change := range outerChangeList.value {
			if change.value.value != nil {
				nmc.cache.Set(outerChangeList.key, change.key, change.value)
			} else {
				nmc.cache.Remove(outerChangeList.key, change.key)
			}
		}
	}

	nmc.transactions.remove(*txKey)
}

func (nmc *NestedMapCache[K1, K2, V]) Clear(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling Clear without initialized cache transaction")
	}

	nmc.transactions.remove(*txKey)
}

func (nmc *NestedMapCache[K1, K2, V]) ClearTransactions() {
	nmc.transactions.transactions = nil
}

func (nmc *NestedMapCache[K1, K2, V]) CheckCache(ctx context.Context) error {
	if err := nmc.checkCacheComplete(ctx); err != nil {
		return fmt.Errorf("error checkCacheComplete: %w", err)
	}

	return nil
}

func (nmc *NestedMapCache[K1, K2, V]) checkCollectionComplete(goCtx context.Context, key1 K1) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	iterator := nmc.Iterator(goCtx, nil, key1)

	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		if keyValue.value.value != nil {
			continue
		}

		before := ctx.GasMeter().GasConsumed()
		key := collections.Join(key1, keyValue.key)
		_, err := nmc.collection.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("could not get key: %v", keyValue.key)
		}
		after := ctx.GasMeter().GasConsumed()

		//if !nmc.valueComparer(*keyValue.value.value, value) {
		//	return fmt.Errorf("differing values for key: %v", keyValue.key)
		//}

		consumption := after - before
		if consumption != keyValue.value.cost {
			return fmt.Errorf("consumption: %v, cache consumption: %v", consumption, keyValue.value.cost)
		}
	}

	return nil
}

func (nmc *NestedMapCache[K1, K2, V]) checkCacheComplete(ctx context.Context) error {
	iterator, err := nmc.collection.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not create iterator: %w", err)
	}

	keyValues, err := iterator.KeyValues()
	if err != nil {
		return err
	}

	for _, keyValue := range keyValues {
		value, has := nmc.cache.Get(keyValue.Key.K1(), keyValue.Key.K2())
		if !has {
			return fmt.Errorf("could not get key: %v", keyValue.Key)
		}

		// in this case the value needed to be loaded from storage, so testing whether they are equal isn't necessary
		if value.value == nil {
			continue
		}

		//if !nmc.valueComparer(keyValue.Value, *value.value) {
		//	return fmt.Errorf("differing values for key: %v", keyValue.Key)
		//}
	}

	return nil
}
