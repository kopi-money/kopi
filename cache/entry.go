package cache

type Entry[V any] struct {
	value *V
	cost  uint64
}

func (e Entry[V]) Value() *V {
	return e.value
}
