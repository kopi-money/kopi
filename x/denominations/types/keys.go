package types

const (
	// ModuleName defines the module name
	ModuleName = "denominations"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_denominations"
)

var (
	ParamsKey = []byte("p_denominations")
)
