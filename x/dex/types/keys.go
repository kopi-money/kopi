package types

import (
	"strconv"
)

const (
	ModuleName = "dex"
	StoreKey   = ModuleName

	PoolDex       = ModuleName
	PoolFees      = "pool_dex_fees"
	PoolOrders    = "pool_orders"
	PoolReserve   = "pool_reserve"
	PoolLiquidity = "pool_liquidity"
	PoolTrade     = "pool_trade"

	KeyLiquidityNextIndex = "LiquidityNextIndex/value"
	KeyOrderNextIndex     = "OrderNextIndex/value"
	KeyLiquidityPairCount = "LiquidityPair/count/"

	KeyPrefixRatio         = "Ratio/value/"
	KeyPrefixOrder         = "Order/value/"
	KeyPrefixLiquiditySum  = "LiquiditySum/value/"
	KeyPrefixLiquidityPair = "LiquidityPair/value/"
	KeyPrefixLiquidity     = "Liquidity/value"
	KeyPrefixTradeAmount   = "TradeAmount/value/"

	MemStoreKey = "mem_dex"
)

var (
	ParamsKey = []byte("p_dex")
)

func Key(p string) []byte {
	return []byte(p)
}

func KeyString(denom string) (key []byte) {
	key = append(key, []byte(denom)...)
	key = append(key, []byte("/")...)

	return key
}

func KeyIndex(id uint64) []byte {
	var key []byte

	idBytes := []byte(strconv.Itoa(int(id)))
	key = append(key, idBytes...)
	key = append(key, []byte("/")...)

	return key
}

func KeyAddressDenom(address, denom string) (key []byte) {
	key = append(key, []byte(address)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(denom)...)
	key = append(key, []byte("/")...)

	return key
}

func KeyDenomIndex(denom string, index uint64) []byte {
	var key []byte

	key = append(key, []byte(denom)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(strconv.Itoa(int(index)))...)
	key = append(key, []byte("/")...)

	return key
}
