package reserve

import (
	"context"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/reserve/keeper"
	"github.com/kopi-money/kopi/x/reserve/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init

	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.SetParams(innerCtx, genState.Params)
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
