package types

const (
	// ModuleName defines the module name
	ModuleName = "tokenfactory"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	KeyPrefixFactoryDenoms = "FactoryDenoms/values/"
)

var (
	ParamsKey = Key("p_tokenfactory")
)

func Key(p string) []byte {
	return []byte(p)
}

func KeyDenom(denom string) (key []byte) {
	key = append(key, []byte(denom)...)
	key = append(key, []byte("/")...)
	return key
}

func KeyDenomAddress(denom, address string) (key []byte) {
	key = append(key, []byte(denom)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(address)...)
	key = append(key, []byte("/")...)
	return key
}
