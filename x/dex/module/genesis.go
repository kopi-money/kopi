package dex

import (
	"context"
	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.InitGenesis(innerCtx, genState)
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	// this line is used by starport scaffolding # genesis/module/export
	return k.ExportGenesis(ctx)
}
