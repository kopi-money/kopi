package cache

import "cosmossdk.io/collections"

var (
	StringComparer       = StringCompare{}
	StringStringComparer = StringStringCompare{}
	StringUInt64Comparer = StringUint64Compare{}
	Uint64Comparer       = Uint64Compare{}
)

type ValueComparer[V any] func(v1, v2 V) bool

type KeyComparer interface {
	Equal(any, any) bool
	Less(any, any) bool
	LTE(any, any) bool
}

type StringCompare struct{}

func (sc StringCompare) Equal(v1, v2 any) bool {
	s1 := v1.(string)
	s2 := v2.(string)
	return s1 == s2
}

func (sc StringCompare) Less(v1, v2 any) bool {
	s1 := v1.(string)
	s2 := v2.(string)
	return s1 < s2
}

func (sc StringCompare) LTE(v1, v2 any) bool {
	s1 := v1.(string)
	s2 := v2.(string)
	return s1 <= s2
}

type Uint64Compare struct{}

func (uc Uint64Compare) Equal(v1, v2 any) bool {
	s1 := v1.(uint64)
	s2 := v2.(uint64)
	return s1 == s2
}

func (uc Uint64Compare) Less(v1, v2 any) bool {
	s1 := v1.(uint64)
	s2 := v2.(uint64)
	return s1 < s2
}

func (uc Uint64Compare) LTE(v1, v2 any) bool {
	s1 := v1.(uint64)
	s2 := v2.(uint64)
	return s1 <= s2
}

type StringStringCompare struct{}

func (ssc StringStringCompare) Equal(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, string])
	p2 := v2.(collections.Pair[string, string])

	if p1.K1() != p2.K1() {
		return false
	}

	return p1.K2() == p2.K2()
}

func (ssc StringStringCompare) Less(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, string])
	p2 := v2.(collections.Pair[string, string])

	if p1.K1() == p2.K1() {
		return p1.K2() < p2.K2()
	}

	return p1.K1() < p2.K1()
}

func (ssc StringStringCompare) LTE(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, string])
	p2 := v2.(collections.Pair[string, string])

	if p1.K1() == p2.K1() {
		return p1.K2() <= p2.K2()
	}

	return p1.K1() <= p2.K1()
}

type StringUint64Compare struct{}

func (suc StringUint64Compare) Equal(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, uint64])
	p2 := v2.(collections.Pair[string, uint64])

	if p1.K1() != p2.K1() {
		return false
	}

	return p1.K2() == p2.K2()
}

func (suc StringUint64Compare) Less(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, uint64])
	p2 := v2.(collections.Pair[string, uint64])

	if p1.K1() == p2.K1() {
		return p1.K2() < p2.K2()
	}

	return p1.K1() < p2.K1()
}

func (suc StringUint64Compare) LTE(v1, v2 any) bool {
	p1 := v1.(collections.Pair[string, uint64])
	p2 := v2.(collections.Pair[string, uint64])

	if p1.K1() == p2.K1() {
		return p1.K2() <= p2.K2()
	}

	return p1.K1() <= p2.K1()
}
