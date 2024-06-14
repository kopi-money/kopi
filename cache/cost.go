package cache

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
)

const (
	ReadCostFlat    uint64 = 1000
	ReadCostPerByte uint64 = 3
)

func CalculateReadCostMap[K, V any](prefix []byte, kc codec.KeyCodec[K], vc codec.ValueCodec[V], key K, value V) uint64 {
	keyBytes, _ := collections.EncodeKeyWithPrefix(prefix, kc, key)
	valueBytes, _ := vc.Encode(value)

	var cost uint64
	cost += ReadCostFlat
	cost += ReadCostPerByte * uint64(len(keyBytes))
	cost += ReadCostPerByte * uint64(len(valueBytes))
	return cost
}

func CalculateReadCostItem[V any](vc codec.ValueCodec[V], value V) uint64 {
	valueBytes, _ := vc.Encode(value)

	var cost uint64
	cost += ReadCostFlat
	cost += ReadCostPerByte * uint64(len(valueBytes))
	return cost
}
