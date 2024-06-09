package types

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		LiquidityList:      []Liquidity{},
		LiquidityPairList:  []LiquidityPair{},
		LiquidityPairCount: 0,
		RatioList:          []Ratio{},
		LiquidityNextIndex: 0,

		LiquiditySumList:  []LiquiditySum{},
		OrderList:         []Order{},
		WalletTradeAmount: []WalletTradeAmount{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
