package cache

import (
	"context"
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Iterator[K, V any] interface {
	Valid() bool
	GetAll() []V
	GetNext() V
	GetNextKeyValue() KeyValue[K, Entry[V]]
	GetAllFromCache() []KeyValue[K, Entry[V]]
}

type Filter[K any] func(k K) bool

type IteratorList[K, V any] struct {
	orderedList *OrderedList[K, Entry[V]]
	currentItem *KeyValue[K, Entry[V]]

	deleteList []K
	filter     Filter[K]

	index    int
	useEmpty bool
}

func (il *IteratorList[K, V]) stepToNextValue() {
	if il.orderedList.Size() == 0 {
		return
	}

	for il.index < il.orderedList.Size() {
		entry := il.orderedList.GetByIndex(il.index)
		il.index++

		if il.filter != nil && !il.filter(entry.key) {
			continue
		}

		if len(il.deleteList) > 0 {
			if il.orderedList.comparer.Equal(il.deleteList[0], entry.key) {
				il.deleteList = il.deleteList[1:]
				continue
			}
		}

		if il.useEmpty || entry.value.value != nil {
			il.currentItem = &entry
			return
		}
	}

	il.currentItem = nil
}

func (il *IteratorList[K, V]) has() bool {
	return il.currentItem != nil
}

func (il *IteratorList[K, V]) key() K {
	return il.currentItem.key
}

func (il *IteratorList[K, V]) next() KeyValue[K, Entry[V]] {
	item := *il.currentItem
	il.stepToNextValue()
	return item
}

func newIterator[K, V any](ctx context.Context, cache, changes *OrderedList[K, Entry[V]], mapCache *MapCache[K, V], deleted []K, rng collections.Ranger[K], filter Filter[K]) Iterator[K, V] {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() == mapCache.currentHeight {
		return newCacheIterator(ctx, cache, changes, mapCache, deleted, filter)
	} else {
		return newCollectionIterator(sdkCtx, mapCache, rng)
	}
}

type CollectionIterator[K, V any] struct {
	iterator collections.Iterator[K, V]
}

func (c CollectionIterator[K, V]) GetAll() (list []V) {
	for c.Valid() {
		list = append(list, c.GetNext())
	}

	return
}

// Probably not the most elegant way to do this
func (c CollectionIterator[K, V]) GetAllFromCache() []KeyValue[K, Entry[V]] {
	panic("implement me")
}

func (c CollectionIterator[K, V]) Valid() bool {
	return c.iterator.Valid()
}

func (c CollectionIterator[K, V]) GetNext() V {
	v, _ := c.iterator.Value()
	c.iterator.Next()
	return v
}

func (c CollectionIterator[K, V]) GetNextKeyValue() KeyValue[K, Entry[V]] {
	c.iterator.Next()
	kv, _ := c.iterator.KeyValue()
	return KeyValue[K, Entry[V]]{
		key: kv.Key,
		value: Entry[V]{
			value: &kv.Value,
			cost:  0,
		},
	}
}

func newCollectionIterator[K, V any](ctx context.Context, mapCache *MapCache[K, V], rng collections.Ranger[K]) Iterator[K, V] {
	iterator, _ := mapCache.collection.Iterate(ctx, rng)
	return &CollectionIterator[K, V]{
		iterator: iterator,
	}
}

type CacheIterator[K, V any] struct {
	ctx            context.Context
	changes        *IteratorList[K, V]
	cache          *IteratorList[K, V]
	mapCache       *MapCache[K, V]
	smallestDelete *K
}

func newCacheIterator[K, V any](ctx context.Context, cache, changes *OrderedList[K, Entry[V]], mapCache *MapCache[K, V], deleted []K, filter Filter[K]) Iterator[K, V] {
	iterator := CacheIterator[K, V]{
		ctx: ctx,
		changes: &IteratorList[K, V]{
			orderedList: changes,
			filter:      filter,
		},
		cache: &IteratorList[K, V]{
			orderedList: cache,
			filter:      filter,
			deleteList:  deleted,
			useEmpty:    true,
		},
		mapCache: mapCache,
	}

	iterator.stepBoth()
	return &iterator
}

func (it *CacheIterator[K, V]) stepBoth() {
	it.changes.stepToNextValue()
	it.cache.stepToNextValue()
}

func (it *CacheIterator[K, V]) Valid() bool {
	return it.changes.currentItem != nil || it.cache.currentItem != nil
}

func (it *CacheIterator[K, V]) GetNext() V {
	next := it.GetNextKeyValue()
	if next.value.value == nil {
		value, _ := it.mapCache.Get(it.ctx, next.key)
		return value
	}

	return *next.value.value
}

func (it *CacheIterator[K, V]) GetNextKeyValue() KeyValue[K, Entry[V]] {
	if it.changes.has() && !it.cache.has() {
		return it.changes.next()
	}

	if !it.changes.has() && it.cache.has() {
		return it.cache.next()
	}

	if it.mapCache.keyComparer.Equal(it.changes.key(), it.cache.key()) {
		it.cache.stepToNextValue()
		return it.changes.next()
	}

	if it.mapCache.keyComparer.Less(it.changes.key(), it.cache.key()) {
		return it.changes.next()
	} else {
		return it.cache.next()
	}
}

func (it *CacheIterator[K, V]) GetAllFromCache() []KeyValue[K, Entry[V]] {
	return it.cache.orderedList.GetAll()
}

func (it *CacheIterator[K, V]) GetAll() (list []V) {
	for it.Valid() {
		list = append(list, it.GetNext())
	}

	return
}
