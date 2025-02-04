package swap

import (
	"context"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/swap/keeper"
	"github.com/kopi-money/kopi/x/swap/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {

		// this line is used by starport scaffolding # genesis/module/init
		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
