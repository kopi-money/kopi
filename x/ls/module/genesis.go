package ls

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/ls/keeper"
	"github.com/kopi-money/kopi/x/ls/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		k.SetUndelegations(innerCtx, genState.Undelegations)
		k.SetDelegationNextIndex(innerCtx, genState.UndelegationsNextIndex)

		if err := k.SetParams(ctx, genState.Params); err != nil {
			return fmt.Errorf("could not set params: %w", err)
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
	genesis.UndelegationsNextIndex = k.GetDelegationNextIndex(ctx)
	genesis.Undelegations = k.GetGenesisUndelegations(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
