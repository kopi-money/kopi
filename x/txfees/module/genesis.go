package txfees

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/txfees/keeper"
	"github.com/kopi-money/kopi/x/txfees/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(_ sdk.Context, _ keeper.Keeper, _ types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(_ sdk.Context, _ keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
