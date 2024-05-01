package types

const (
	ModuleName = "mm"
	StoreKey   = ModuleName

	PoolCollateral = "pool_collateral"
	PoolVault      = "pool_vault"
	PoolRedemption = "pool_redemption"

	KeyPrefixLoansIndex  = "Loans/index/"
	KeyPrefixLoans       = "Loans/value/"
	KeyPrefixCollaterals = "Collaterals/value/"
	KeyPrefixRedemptions = "Redemptions/value/"
)

var (
	ParamsKey = []byte("p_mm")
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
