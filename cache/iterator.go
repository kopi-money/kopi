package cache

import (
	"context"

	"cosmossdk.io/collections"
)

type ValueGetter[K, V any] func(K) V

type Iterator[K ordered, V any] interface {
	Valid() bool
	GetAll() []V
	GetNext() V
	GetNextKeyValue() KeyValue[K, Entry[V]]
	GetAllFromCache() []KeyValue[K, Entry[V]]
}

type Filter[K any] func(k K) bool

type IteratorList[K ordered, V any] struct {
	orderedList *OrderedList[K, Entry[V]]
	currentItem *KeyValue[K, Entry[V]]

	deleteList []K

	index    int
	useEmpty bool
}

func (il *IteratorList[K, V]) has() bool {
	return il.currentItem != nil
}

func (il *IteratorList[K, V]) hasRemovedItem() bool {
	return il.currentItem != nil && il.currentItem.value.value == nil
}

func (il *IteratorList[K, V]) atEnd() bool {
	return il.index >= il.orderedList.Size()
}

func (il *IteratorList[K, V]) stepToNextValue() {
	if il.orderedList.Size() == 0 {
		return
	}

	for il.index < il.orderedList.Size() {
		entry := il.orderedList.GetByIndex(il.index)
		il.index++

		if len(il.deleteList) > 0 {
			if il.deleteList[0] == entry.key {
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
	return
}

func (il *IteratorList[K, V]) key() K {
	return il.currentItem.key
}

func (il *IteratorList[K, V]) next() KeyValue[K, Entry[V]] {
	item := *il.currentItem
	il.stepToNextValue()

	return item
}

func useCache(ctx context.Context, cacheHeight int64) bool {
	requestedHeight := getCurrentHeight(ctx)
	if requestedHeight == 0 {
		requestedHeight = cacheHeight
	}

	return requestedHeight == cacheHeight
}

func newIterator[K ordered, V any](ctx context.Context, cache, changes *OrderedList[K, Entry[V]], valueGetter ValueGetter[K, V], deleted []K, cci CreateCollectionIterator[K, V], currentHeight int64) Iterator[K, V] {
	if useCache(ctx, currentHeight) {
		return newCacheIterator(ctx, cache, changes, valueGetter, deleted)
	} else {
		return cci()
	}
}

type CollectionIterator[K ordered, V any] struct {
	iterator collections.Iterator[K, V]
}

func (c CollectionIterator[K, V]) GetAll() (list []V) {
	for c.Valid() {
		list = append(list, c.GetNext())
	}

	return
}

// Probably not the most elegant way to do this...
func (c CollectionIterator[K, V]) GetAllFromCache() (entries []KeyValue[K, Entry[V]]) {
	for c.iterator.Valid() {
		kv, _ := c.iterator.KeyValue()
		c.iterator.Next()

		entries = append(entries, KeyValue[K, Entry[V]]{
			key: kv.Key,
			value: Entry[V]{
				value: &kv.Value,
				cost:  0,
			},
		})
	}

	return
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

type CacheIterator[K ordered, V any] struct {
	ctx            context.Context
	changes        *IteratorList[K, V]
	cache          *IteratorList[K, V]
	valueGetter    ValueGetter[K, V]
	smallestDelete *K
}

func newCacheIterator[K ordered, V any](ctx context.Context, cache, changes *OrderedList[K, Entry[V]], valueGetter ValueGetter[K, V], deleted []K) Iterator[K, V] {
	iterator := CacheIterator[K, V]{
		ctx: ctx,
		changes: &IteratorList[K, V]{
			orderedList: changes,
			useEmpty:    true,
		},
		cache: &IteratorList[K, V]{
			orderedList: cache,
			deleteList:  deleted,
		},
		valueGetter: valueGetter,
	}

	iterator.stepToFirst()
	return &iterator
}

// stepToFirst is called at the beginning and sets the two lists to the initial value. We have to cover the edge case
// where the first common entry is a delete entry. In that case, both lists have to step further
func (it *CacheIterator[K, V]) stepToFirst() {
	for {
		it.changes.stepToNextValue()
		it.cache.stepToNextValue()

		// If changes is at a deleted item but cache doesn't have items, changes can go another step
		if !it.cache.has() && it.changes.hasRemovedItem() {
			continue
		}

		// If both lists are at an item for which the changes list says it is deleted, both lists have to another step
		if !it.currentCacheItemIsDeleted() {
			break
		}
	}
}

func (it *CacheIterator[K, V]) currentCacheItemIsDeleted() bool {
	if it.changes.currentItem != nil && it.cache.currentItem != nil {
		if it.changes.currentItem.key == it.cache.currentItem.key {
			return it.changes.currentItem.value.value == nil
		}
	}

	return false
}

func (it *CacheIterator[K, V]) Valid() bool {
	if it.changes.currentItem != nil && it.changes.currentItem.value.value != nil {
		return true
	}

	if it.cache.currentItem != nil {
		return true
	}

	return false
}

func (it *CacheIterator[K, V]) GetNext() V {
	next := it.GetNextKeyValue()
	if next.value.value == nil {
		return it.valueGetter(next.key)
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

	if it.changes.key() == it.cache.key() {
		it.cache.stepToNextValue()
		return it.changes.next()
	}

	if it.changes.key() < it.cache.key() {
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
