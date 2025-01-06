package cache

import "cosmossdk.io/collections"

type NestedIterator[K1, K2 ordered, V any] struct {
	iterator collections.Iterator[collections.Pair[K1, K2], V]
}

func (ni NestedIterator[K1, K2, V]) Valid() bool {
	return ni.iterator.Valid()
}

func (ni NestedIterator[K1, K2, V]) GetAll() (list []V) {
	for ni.Valid() {
		list = append(list, ni.GetNext())
	}

	return
}

func (ni NestedIterator[K1, K2, V]) GetNext() V {
	v, _ := ni.iterator.Value()
	ni.iterator.Next()
	return v
}

func (ni NestedIterator[K1, K2, V]) GetNextKeyValue() KeyValue[K2, Entry[V]] {
	kv, _ := ni.iterator.KeyValue()
	ni.iterator.Next()

	return KeyValue[K2, Entry[V]]{
		key: kv.Key.K2(),
		value: Entry[V]{
			value: &kv.Value,
			cost:  0,
		},
	}
}

func (ni NestedIterator[K1, K2, V]) GetAllFromCache() []KeyValue[K2, Entry[V]] {
	panic("implement me")
}
