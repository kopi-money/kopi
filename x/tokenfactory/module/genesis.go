package tokenfactory

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/tokenfactory/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		for _, pool := range genState.LiquidityPools {
			k.SetGenesisLiquidityPool(innerCtx, pool)
		}

		for _, conversion := range genState.TokenConversions {
			k.SetGenesisTokenConversion(innerCtx, conversion)
		}

		// this line is used by starport scaffolding # genesis/module/init
		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return fmt.Errorf("set params: %w", err)
		}

		for _, denom := range genState.FactoryDenoms {
			k.SetDenom(innerCtx, denom)
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
	genesis.FactoryDenoms = k.GetAllDenoms(ctx)
	genesis.LiquidityPools = k.GetGenesisLiquidityPools(ctx)
	genesis.TokenConversions = k.GetGenesisTokenConversions(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
