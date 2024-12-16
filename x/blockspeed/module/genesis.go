package blockspeed

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/blockspeed/keeper"
	"github.com/kopi-money/kopi/x/blockspeed/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return fmt.Errorf("could not set params: %w", err)
		}

		if genState.Blockspeed == nil {
			genState.Blockspeed = &types.Blockspeed{
				PreviousTimestamp: 0,
				AverageTime:       math.LegacyOneDec(),
			}
		}

		k.SetBlockspeed(innerCtx, *genState.Blockspeed)

		return nil
	}); err != nil {
		panic(err)
	}
	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	bs := k.GetBlockspeed(ctx)

	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Blockspeed = &bs

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
