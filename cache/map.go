package cache

import (
	"context"
	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sync"
)

type Entry[V any] struct {
	// The actual value from storage. THe value is nil when it has been removed
	value *V

	// The gas cost for reading this value
	consumption uint64

	// exists indicates whether for a given key there is a value in storage
	exists bool
}

type MapTransaction[K, V any] struct {
	key     string
	changes *OrderedList[K, Entry[V]]

	removals []K
	comparer KeyComparer
}

func (mt *MapTransaction[K, V]) remove(key K) {
	mt.addToRemovals(key)
	mt.changes.Set(KeyValue[K, Entry[V]]{
		key: key,
		value: Entry[V]{
			value:  nil,
			exists: false,
		},
	})
}

func (mt *MapTransaction[K, V]) addToRemovals(key K) {
	for _, k := range mt.removals {
		if mt.comparer.Equal(k, key) {
			return
		}
	}

	mt.removals = append(mt.removals, key)
}

func (mt *MapTransaction[K, V]) set(keyValue KeyValue[K, Entry[V]]) {
	mt.removeFromRemovals(keyValue.key)
	mt.changes.Set(keyValue)
}

func (mt *MapTransaction[K, V]) removeFromRemovals(key K) {
	index := -1
	for i, k := range mt.removals {
		if mt.comparer.Equal(k, key) {
			index = i
			break
		}
	}

	if index != -1 {
		mt.removals = append(mt.removals[:index], mt.removals[index+1:]...)
	}
}

type MapTransactions[K, V any] struct {
	sync.RWMutex

	comparer     KeyComparer
	transactions []*MapTransaction[K, V]
}

func (mt *MapTransactions[K, V]) Get(key string) *MapTransaction[K, V] {
	if key == "" {
		return nil
	}

	mapTransaction := mt.get(key)
	if mapTransaction != nil {
		return mapTransaction
	}

	mapTransaction = &MapTransaction[K, V]{
		key:      key,
		changes:  newOrderedList[K, Entry[V]](mt.comparer),
		comparer: mt.comparer,
	}
	mt.set(mapTransaction)

	return mapTransaction
}

func (mt *MapTransactions[K, V]) get(key string) *MapTransaction[K, V] {
	mt.RLock()
	defer mt.RUnlock()

	for _, mapTransaction := range mt.transactions {
		if mapTransaction.key == key {
			return mapTransaction
		}
	}

	return nil
}

func (mt *MapTransactions[K, V]) set(mapTransaction *MapTransaction[K, V]) {
	mt.Lock()
	defer mt.Unlock()

	mt.transactions = append(mt.transactions, mapTransaction)
}

func (mt *MapTransactions[K, V]) remove(key string) {
	mt.Lock()
	defer mt.Unlock()

	index := -1
	for i, mapTransaction := range mt.transactions {
		if mapTransaction.key == key {
			index = i
			break
		}
	}

	if index != -1 {
		mt.transactions = append(mt.transactions[:index], mt.transactions[index+1:]...)
	}
}

func getEntry[V any](goCtx context.Context, entry Entry[V], consume bool) (V, bool) {
	if consume {
		ctx := sdk.UnwrapSDKContext(goCtx)
		ctx.GasMeter().ConsumeGas(entry.consumption, "")
	}

	if entry.value != nil {
		return *entry.value, true
	}

	var value V
	return value, false
}

type MapCache[K, V any] struct {
	sync.Mutex

	collection    collections.Map[K, V]
	cache         *OrderedList[K, Entry[V]]
	transactions  *MapTransactions[K, V]
	keyComparer   KeyComparer
	valueComparer ValueComparer[V]
	initialized   bool
}

func NewCacheMap[K, V any](collection collections.Map[K, V], caches *Caches, keyComparer KeyComparer, valueComparer ValueComparer[V]) *MapCache[K, V] {
	mc := &MapCache[K, V]{
		cache:         newOrderedList[K, Entry[V]](keyComparer),
		transactions:  &MapTransactions[K, V]{comparer: keyComparer},
		collection:    collection,
		keyComparer:   keyComparer,
		valueComparer: valueComparer,
	}

	*caches = append(*caches, mc)
	return mc
}

func (mc *MapCache[K, V]) Initialize(goCtx context.Context) error {
	if mc.initialized {
		return nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	iterator, err := mc.collection.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not create collection iterator")
	}

	var (
		entries []KeyValue[K, Entry[V]]
		key     K
	)

	for ; iterator.Valid(); iterator.Next() {
		key, err = iterator.Key()
		if err != nil {
			return errors.Wrap(err, "could not get KeyValue")
		}

		entry := mc.loadFromStorage(ctx, key, false)
		entries = append(entries, KeyValue[K, Entry[V]]{key: key, value: entry})
	}

	mc.cache.set(entries)
	mc.initialized = true

	return nil
}

func (mc *MapCache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	if !mc.initialized {
		_ = mc.Initialize(ctx)
	}

	txKey, hasTX := getTXKey(ctx)
	if hasTX {
		mapTransaction := mc.transactions.Get(txKey)
		change, has := mapTransaction.changes.Get(key)
		if has {
			return getEntry(ctx, change, true)
		}
	}

	entry, has := mc.cache.Get(key)
	if has && entry.exists && entry.value != nil {
		return getEntry(ctx, entry, hasTX)
	}

	entry = mc.loadFromStorage(ctx, key, false)
	mc.cache.Set(KeyValue[K, Entry[V]]{
		key:   key,
		value: entry,
	})

	return getEntry(ctx, entry, false)
}

func (mc *MapCache[K, V]) loadFromStorage(goCtx context.Context, key K, preventGasConsumption bool) Entry[V] {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if preventGasConsumption {
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	before := ctx.GasMeter().GasConsumed()
	value, err := mc.collection.Get(ctx, key)
	after := ctx.GasMeter().GasConsumed()

	entry := Entry[V]{
		exists:      true,
		consumption: after - before,
	}

	if err == nil {
		entry.value = &value
	}

	return entry
}

func (mc *MapCache[K, V]) Set(ctx context.Context, key K, value V) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling set without initialized cache transaction")
	}

	mapTransaction := mc.transactions.Get(txKey)
	mapTransaction.set(KeyValue[K, Entry[V]]{
		key: key,
		value: Entry[V]{
			value:  &value,
			exists: true,
		},
	})
}

func (mc *MapCache[K, V]) Remove(ctx context.Context, key K) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling set without initialized cache transaction")
	}

	mapTransaction := mc.transactions.Get(txKey)
	mapTransaction.remove(key)
}

// Iterator returns an iterator which contains a list of all keys. Since the cache doesn't know about all keys, they
// have to be loaded from storage first. Then interim changes to the data have to be applied to the keys, i.e.
// adding new ones or removes those that have been deleted. If new keys are added, the list has to be sorted once more.
func (mc *MapCache[K, V]) Iterator(ctx context.Context, filters ...Filter[K]) *Iterator[K, V] {
	var changes *OrderedList[K, Entry[V]]
	var removals []K

	txKey, has := getTXKey(ctx)
	if has {
		mapTransaction := mc.transactions.Get(txKey)
		changes = mapTransaction.changes
		removals = mapTransaction.removals
	} else {
		changes = newOrderedList[K, Entry[V]](mc.keyComparer)
	}

	return newIterator(ctx, mc.cache, changes, mc, removals, filters...)
}

func (mc *MapCache[K, V]) Size() int {
	return mc.cache.Size()
}

func (mc *MapCache[K, V]) CommitToDB(ctx context.Context) error {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling commit without initialized cache transaction")
	}

	for _, change := range mc.transactions.Get(txKey).changes.GetAll() {
		if change.value.value != nil {
			if err := mc.collection.Set(ctx, change.key, *change.value.value); err != nil {
				return errors.Wrap(err, "could not add value to collection")
			}
		} else {
			if err := mc.collection.Remove(ctx, change.key); err != nil {
				return errors.Wrap(err, "could not remove value from collection")
			}
		}
	}

	return nil
}

func (mc *MapCache[K, V]) CommitToCache(ctx context.Context) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling commit without initialized cache transaction")
	}

	for _, change := range mc.transactions.Get(txKey).changes.GetAll() {
		if change.value.value == nil {
			mc.cache.Remove(change.key)
		} else {
			mc.cache.Set(KeyValue[K, Entry[V]]{
				key: change.key,
				value: Entry[V]{
					value:  nil,
					exists: change.value.value != nil,
				},
			})
		}
	}

	mc.transactions.remove(txKey)
}

func (mc *MapCache[K, V]) Rollback(ctx context.Context) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling rollback without initialized cache transaction")
	}

	mc.transactions.remove(txKey)
}

func (mc *MapCache[K, V]) ClearTransactions() {
	mc.transactions.transactions = nil
}

func (mc *MapCache[K, V]) CheckCache(ctx context.Context) error {
	if err := mc.checkCacheComplete(ctx); err != nil {
		return errors.Wrap(err, "error checkCacheComplete")
	}

	if err := mc.checkCacheComplete(ctx); err != nil {
		return errors.Wrap(err, "error checkCacheComplete")
	}

	return nil
}

func (mc *MapCache[K, V]) checkCollectionComplete(ctx context.Context) error {
	iterator := mc.Iterator(ctx)

	var keyValue KeyValue[K, Entry[V]]
	for iterator.Valid() {
		keyValue = iterator.GetNextKeyValue()
		if !keyValue.value.exists {
			continue
		}

		value, err := mc.collection.Get(ctx, keyValue.key)
		if err != nil {
			return fmt.Errorf("could not get key: %v", keyValue.key)
		}

		if !mc.valueComparer(*keyValue.value.value, value) {
			return fmt.Errorf("differing values for key: %v", keyValue.key)
		}
	}

	return nil
}

func (mc *MapCache[K, V]) checkCacheComplete(ctx context.Context) error {
	iterator, err := mc.collection.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not create iterator")
	}

	keyValues, err := iterator.KeyValues()
	if err != nil {
		return err
	}

	for _, keyValue := range keyValues {
		value, has := mc.cache.Get(keyValue.Key)
		if !has {
			return fmt.Errorf("could not get key: %v", keyValue.Key)
		}

		// in this case the value needed to be loaded from storage, so testing whether they are equal isn't necessary
		if value.value == nil {
			continue
		}

		if value.exists && !mc.valueComparer(keyValue.Value, *value.value) {
			return fmt.Errorf("differing values for key: %v", keyValue.Key)
		}
	}

	return nil
}
