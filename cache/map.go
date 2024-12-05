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

type CreateCollectionIterator[K ordered, V any] func() Iterator[K, V]

type CollectionMap[K, V any] interface {
	Get(context.Context, K) (V, error)
	Iterate(context.Context, collections.Ranger[K]) (collections.Iterator[K, V], error)
	IterateRaw(ctx context.Context, start, end []byte, order collections.Order) (collections.Iterator[K, V], error)
	Set(context.Context, K, V) error
	Remove(context.Context, K) error
	GetName() string
}

type Entry[V any] struct {
	value *V
	cost  uint64
}

func (e Entry[V]) Value() *V {
	return e.value
}

type MapTransaction[K ordered, V any] struct {
	key      TXKey
	changes  *OrderedList[K, Entry[V]]
	previous *OrderedList[K, Entry[V]]

	removals []K
}

func (mt *MapTransaction[K, V]) remove(key K, previous Entry[V]) {
	mt.addToRemovals(key)
	mt.changes.Set(KeyValue[K, Entry[V]]{
		key:   key,
		value: Entry[V]{},
	})

	if has := mt.previous.Has(key); !has {
		mt.previous.Set(KeyValue[K, Entry[V]]{
			key:   key,
			value: previous,
		})
	}
}

func (mt *MapTransaction[K, V]) addToRemovals(key K) {
	for _, k := range mt.removals {
		if k == key {
			return
		}
	}

	mt.removals = append(mt.removals, key)
}

func (mt *MapTransaction[K, V]) set(keyValue KeyValue[K, Entry[V]], previous Entry[V]) {
	mt.removeFromRemovals(keyValue.key)
	mt.changes.Set(keyValue)

	if has := mt.previous.Has(keyValue.key); !has {
		mt.previous.Set(KeyValue[K, Entry[V]]{
			key:   keyValue.key,
			value: previous,
		})
	}
}

func (mt *MapTransaction[K, V]) removeFromRemovals(key K) {
	index := -1
	for i, k := range mt.removals {
		if k == key {
			index = i
			break
		}
	}

	if index != -1 {
		mt.removals = append(mt.removals[:index], mt.removals[index+1:]...)
	}
}

type MapTransactions[K ordered, V any] struct {
	sync.RWMutex

	transactions []*MapTransaction[K, V]
}

func (mt *MapTransactions[K, V]) Get(key TXKey) *MapTransaction[K, V] {
	mapTransaction := mt.get(key)
	if mapTransaction != nil {
		return mapTransaction
	}

	mapTransaction = &MapTransaction[K, V]{
		key:      key,
		changes:  &OrderedList[K, Entry[V]]{},
		previous: &OrderedList[K, Entry[V]]{},
	}
	mt.set(mapTransaction)

	return mapTransaction
}

func (mt *MapTransactions[K, V]) get(key TXKey) *MapTransaction[K, V] {
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

func (mt *MapTransactions[K, V]) remove(key TXKey) {
	mt.Lock()
	defer mt.Unlock()

	index := -1
	for i, mapTransaction := range mt.transactions {
		if mapTransaction.key.equals(key) {
			index = i
			break
		}
	}

	if index != -1 {
		mt.transactions = append(mt.transactions[:index], mt.transactions[index+1:]...)
	}
}

func getEntry[V any](ctx context.Context, entry Entry[V]) (V, bool) {
	if entry.value != nil {
		if sdkCtx, ok := ctx.(sdk.Context); ok {
			sdkCtx.GasMeter().ConsumeGas(entry.cost, "")
		}

		return *entry.value, true
	}

	var value V
	return value, false
}

func hasEntry[V any](ctx context.Context, entry Entry[V]) bool {
	if entry.value != nil {
		if sdkCtx, ok := ctx.(sdk.Context); ok {
			sdkCtx.GasMeter().ConsumeGas(entry.cost, "")
		}

		return true
	}

	return false
}

type MapCache[K ordered, V any] struct {
	sync.Mutex

	kc     codec.KeyCodec[K]
	vc     codec.ValueCodec[V]
	prefix []byte

	collection    CollectionMap[K, V]
	cache         *OrderedList[K, Entry[V]]
	transactions  *MapTransactions[K, V]
	initialized   bool
	currentHeight int64
}

func (mc *MapCache[K, V]) IterateRaw(ctx context.Context, start, end []byte, order collections.Order) (collections.Iterator[K, V], error) {
	return mc.collection.IterateRaw(ctx, start, end, order)
}

func (mc *MapCache[K, V]) KeyCodec() codec.KeyCodec[K] {
	//TODO implement me
	panic("implement me")
}

func NewMapCache[K ordered, V any](sb *collections.SchemaBuilder, prefix []byte, name string, kc codec.KeyCodec[K], vc codec.ValueCodec[V], caches *Caches) *MapCache[K, V] {
	mc := &MapCache[K, V]{
		kc:     kc,
		vc:     vc,
		prefix: prefix,

		cache:        &OrderedList[K, Entry[V]]{},
		transactions: &MapTransactions[K, V]{},
		collection:   collections.NewMap(sb, prefix, name, kc, vc),
	}

	*caches = append(*caches, mc)
	return mc
}

func (mc *MapCache[K, V]) NumRunningTransactions() int {
	return len(mc.transactions.transactions)
}

func (mc *MapCache[K, V]) Initialize(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	mc.currentHeight = sdkCtx.BlockHeight()

	if mc.initialized {
		return nil
	}

	iterator, err := mc.collection.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not create collection iterator: %w", err)
	}

	var key K
	for ; iterator.Valid(); iterator.Next() {
		key, err = iterator.Key()
		if err != nil {
			return fmt.Errorf("could not get key: %w", err)
		}

		entry, has := mc.loadFromStorage(ctx, key)
		if has {
			mc.cache.Set(KeyValue[K, Entry[V]]{key: key, value: entry})
		}
	}

	mc.initialized = true

	return nil
}

func (mc *MapCache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := mc.transactions.Get(*txKey)
		change, has := mapTransaction.changes.Get(key)
		if has {
			return getEntry(ctx, change)
		}
	}

	requestedHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if requestedHeight != mc.currentHeight {
		entry, has := mc.loadFromStorage(ctx, key)
		if has {
			return *entry.value, true
		} else {
			var v V
			return v, false
		}
	}

	entry, has := mc.cache.Get(key)
	if has && entry.value != nil {
		return getEntry(ctx, entry)
	}

	var v V
	return v, false
}

func (mc *MapCache[K, V]) loadFromStorage(ctx context.Context, key K) (Entry[V], bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	gasMeter := sdkCtx.GasMeter()
	ctx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	value, err := mc.collection.Get(ctx, key)
	ctx = sdkCtx.WithGasMeter(gasMeter)

	if err != nil {
		return Entry[V]{}, false
	}

	return Entry[V]{
		value: &value,
		cost:  CalculateReadCostMap(mc.prefix, mc.kc, mc.vc, key, value),
	}, true
}

func (mc *MapCache[K, V]) Set(ctx context.Context, key K, value V) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling set without initialized cache transaction")
	}

	previous, has := mc.cache.Get(key)
	if !has {
		previous = Entry[V]{}
	}

	newEntry := KeyValue[K, Entry[V]]{
		key: key,
		value: Entry[V]{
			value: &value,
			cost:  CalculateReadCostMap(mc.prefix, mc.kc, mc.vc, key, value),
		},
	}

	mapTransaction := mc.transactions.Get(*txKey)
	mapTransaction.set(newEntry, previous)
}

func (mc *MapCache[K, V]) Remove(ctx context.Context, key K) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling set without initialized cache transaction")
	}

	previous, has := mc.cache.Get(key)
	if !has {
		previous = Entry[V]{}
	}

	mapTransaction := mc.transactions.Get(*txKey)
	mapTransaction.remove(key, previous)
}

// Iterator returns an iterator which contains a list of all keys. Since the cache doesn't know about all keys, they
// have to be loaded from storage first. Then interim changes to the data have to be applied to the keys, i.e.
// adding new ones or removes those that have been deleted. If new keys are added, the list has to be sorted once more.
func (mc *MapCache[K, V]) Iterator(ctx context.Context, rng collections.Ranger[K]) Iterator[K, V] {
	var changes *OrderedList[K, Entry[V]]
	var removals []K

	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := mc.transactions.Get(*txKey)
		changes = mapTransaction.changes
		removals = mapTransaction.removals
	} else {
		changes = &OrderedList[K, Entry[V]]{}
	}

	valueGetter := func(key K) V {
		v, _ := mc.Get(ctx, key)
		return v
	}

	createIterator := func() Iterator[K, V] {
		iterator, _ := mc.collection.Iterate(ctx, rng)
		return &CollectionIterator[K, V]{
			iterator: iterator,
		}
	}

	return newIterator(ctx, mc.cache, changes, valueGetter, removals, createIterator, mc.currentHeight)
}

func (mc *MapCache[K, V]) CollectionIterator(ctx context.Context, rng collections.Ranger[K]) (collections.Iterator[K, V], error) {
	return mc.collection.Iterate(ctx, rng)
}

func (mc *MapCache[K, V]) CacheIterator(ctx context.Context) Iterator[K, V] {
	var changes *OrderedList[K, Entry[V]]
	var removals []K

	txKey := getTXKey(ctx)
	if txKey != nil {
		mapTransaction := mc.transactions.Get(*txKey)
		changes = mapTransaction.changes
		removals = mapTransaction.removals
	} else {
		changes = &OrderedList[K, Entry[V]]{}
	}

	valueGetter := func(key K) V {
		v, _ := mc.Get(ctx, key)
		return v
	}

	return newCacheIterator(ctx, mc.cache, changes, valueGetter, removals)
}

func (mc *MapCache[K, V]) Size() int {
	return mc.cache.Size()
}

func (mc *MapCache[K, V]) CommitToDB(ctx context.Context) error {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	for _, change := range mc.transactions.Get(*txKey).changes.GetAll() {
		if change.value.value != nil {
			if err := mc.collection.Set(ctx, change.key, *change.value.value); err != nil {
				return fmt.Errorf("could not add value to collection: %w", err)
			}
		} else {
			if err := mc.collection.Remove(ctx, change.key); err != nil {
				return fmt.Errorf("could not remove value from collection: %w", err)
			}
		}
	}

	return nil
}

func (mc *MapCache[K, V]) Rollback(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	// Setting an infinite gas meter because we never want the following actions to fail due to out of gas reasons.
	if sdkCtx, ok := ctx.(sdk.Context); ok {
		sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		ctx = sdkCtx
	}

	previous := mc.transactions.Get(*txKey).previous.GetAll()
	for _, change := range previous {
		if change.value.value != nil {
			_ = mc.collection.Set(ctx, change.key, *change.value.value)
		} else {
			_ = mc.collection.Remove(ctx, change.key)
		}
	}
}

func (mc *MapCache[K, V]) CommitToCache(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling commit without initialized cache transaction")
	}

	for _, change := range mc.transactions.Get(*txKey).changes.GetAll() {
		if change.value.value != nil {
			mc.cache.Set(KeyValue[K, Entry[V]]{
				key:   change.key,
				value: change.value,
			})
		} else {
			mc.cache.Remove(change.key)
		}
	}

	mc.transactions.remove(*txKey)
}

func (mc *MapCache[K, V]) Clear(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("calling Clear without initialized cache transaction")
	}

	mc.transactions.remove(*txKey)
}

func (mc *MapCache[K, V]) ClearTransactions() {
	mc.transactions.transactions = nil
}

func (mc *MapCache[K, V]) CheckCache(ctx context.Context) error {
	if err := mc.checkCacheComplete(ctx); err != nil {
		return fmt.Errorf("error checkCacheComplete: %w", err)
	}

	return nil
}

func (mc *MapCache[K, V]) checkCollectionComplete(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	iterator := mc.Iterator(goCtx, nil)

	var keyValue KeyValue[K, Entry[V]]
	for iterator.Valid() {
		keyValue = iterator.GetNextKeyValue()
		if keyValue.value.value != nil {
			continue
		}

		before := ctx.GasMeter().GasConsumed()
		_, err := mc.collection.Get(ctx, keyValue.key)
		if err != nil {
			return fmt.Errorf("could not get key: %v", keyValue.key)
		}
		after := ctx.GasMeter().GasConsumed()

		//if !mc.valueComparer(*keyValue.value.value, value) {
		//	return fmt.Errorf("differing values for key: %v", keyValue.key)
		//}

		consumption := after - before
		if consumption != keyValue.value.cost {
			return fmt.Errorf("consumption: %v, cache consumption: %v", consumption, keyValue.value.cost)
		}
	}

	return nil
}

func (mc *MapCache[K, V]) checkCacheComplete(ctx context.Context) error {
	iterator, err := mc.collection.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not create iterator: %w", err)
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

		//if !mc.valueComparer(keyValue.Value, *value.value) {
		//	return fmt.Errorf("differing values for key: %v", keyValue.Key)
		//}
	}

	return nil
}
