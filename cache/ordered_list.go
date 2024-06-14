package cache

import (
	"sort"
)

type KeyValue[K, V any] struct {
	key   K
	value V
}

func (kv KeyValue[K, V]) Value() V {
	return kv.value
}

type OrderedList[K, V any] struct {
	comparer KeyComparer
	list     []KeyValue[K, V]
}

func newOrderedList[K, V any](comparer KeyComparer) *OrderedList[K, V] {
	return &OrderedList[K, V]{
		comparer: comparer,
		list:     []KeyValue[K, V]{},
	}
}

func (ol *OrderedList[K, V]) Size() int {
	return len(ol.list)
}

func (ol *OrderedList[K, V]) Clear() {
	ol.list = nil
}

func (ol *OrderedList[K, V]) GetByIndex(index int) KeyValue[K, V] {
	return ol.list[index]
}

func (ol *OrderedList[K, V]) Has(key K) bool {
	_, has := ol.GetIndex(key)
	return has
}

func (ol *OrderedList[K, V]) Get(key K) (V, bool) {
	listIndex, has := ol.GetIndex(key)
	if !has {
		var v V
		return v, false
	}

	return ol.list[listIndex].value, true
}

func (ol *OrderedList[K, V]) GetAll() (filtered []KeyValue[K, V]) {
	return ol.list
}

func (ol *OrderedList[K, V]) GetKeys() (keys []K) {
	for _, entry := range ol.list {
		keys = append(keys, entry.key)
	}

	return
}

func (ol *OrderedList[K, V]) GetIndex(key K) (int, bool) {
	low, high := 0, len(ol.list)-1

	for low <= high {
		mid := low + (high-low)/2

		if ol.comparer.Equal(ol.list[mid].key, key) {
			return mid, true
		}
		if ol.comparer.Less(ol.list[mid].key, key) {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return 0, false
}

func (ol *OrderedList[K, V]) Remove(key K) {
	listIndex, has := ol.GetIndex(key)
	if has {
		ol.list = append(ol.list[:listIndex], ol.list[listIndex+1:]...)
	}
}

func (ol *OrderedList[K, V]) Set(keyValue KeyValue[K, V]) {
	listIndex, has := ol.GetIndex(keyValue.key)
	entry := KeyValue[K, V]{keyValue.key, keyValue.value}

	if has {
		ol.list[listIndex] = entry
	} else {
		ol.list = append(ol.list, entry)
		ol.sort()
	}
}

func (ol *OrderedList[K, V]) set(keyValues []KeyValue[K, V]) {
	ol.list = nil
	for _, keyValue := range keyValues {
		ol.list = append(ol.list, keyValue)
	}

	ol.sort()
}

func (ol *OrderedList[K, V]) sort() {
	sort.Slice(ol.list, func(i, j int) bool {
		return ol.comparer.Less(ol.list[i].key, ol.list[j].key)
	})
}
