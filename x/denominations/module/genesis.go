package denominations

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/denominations/keeper"
	"github.com/kopi-money/kopi/x/denominations/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(goCtx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		// this line is used by starport scaffolding # genesis/module/init
		if err := k.SetParams(ctx, genState.Params); err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
