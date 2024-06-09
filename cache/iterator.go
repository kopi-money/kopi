package cache

import "context"

type Filter[K any] func(k K) bool

type IteratorList[K, V any] struct {
	orderedList *OrderedList[K, Entry[V]]
	currentItem *KeyValue[K, Entry[V]]

	deleteList []K
	filters    []Filter[K]

	index int
}

func (il *IteratorList[K, V]) stepToNextValue() {
	if il.orderedList.Size() == 0 {
		return
	}

outer:
	for il.index < il.orderedList.Size() {
		entry := il.orderedList.GetByIndex(il.index)
		il.index++

		for _, filter := range il.filters {
			if !filter(entry.key) {
				continue outer
			}
		}

		if len(il.deleteList) > 0 {
			if il.orderedList.comparer.Equal(il.deleteList[0], entry.key) {
				il.deleteList = il.deleteList[1:]
				continue
			}
		}

		if entry.value.exists {
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

type Iterator[K, V any] struct {
	ctx            context.Context
	changes        *IteratorList[K, V]
	cache          *IteratorList[K, V]
	mapCache       *MapCache[K, V]
	smallestDelete *K
}

func newIterator[K, V any](ctx context.Context, cache, changes *OrderedList[K, Entry[V]], mapCache *MapCache[K, V], deleted []K, filters ...Filter[K]) *Iterator[K, V] {
	iterator := Iterator[K, V]{
		ctx: ctx,
		changes: &IteratorList[K, V]{
			orderedList: changes,
			filters:     filters,
		},
		cache: &IteratorList[K, V]{
			orderedList: cache,
			filters:     filters,
			deleteList:  deleted,
		},
		mapCache: mapCache,
	}

	iterator.stepBoth()
	return &iterator
}

func (it *Iterator[K, V]) stepBoth() {
	it.changes.stepToNextValue()
	it.cache.stepToNextValue()
}

func (it *Iterator[K, V]) Valid() bool {
	return it.changes.currentItem != nil || it.cache.currentItem != nil
}

func (it *Iterator[K, V]) GetNext() V {
	next := it.GetNextKeyValue()
	if next.value.value == nil {
		value, _ := it.mapCache.Get(it.ctx, next.key)
		return value
	}

	return *next.value.value
}

func (it *Iterator[K, V]) GetNextKeyValue() KeyValue[K, Entry[V]] {
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

func (it *Iterator[K, V]) GetAll() (list []V) {
	for it.Valid() {
		list = append(list, it.GetNext())
	}

	return
}
