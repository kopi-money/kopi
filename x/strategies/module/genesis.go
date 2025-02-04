package arbitrage

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/strategies/keeper"
	"github.com/kopi-money/kopi/x/strategies/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init

	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return fmt.Errorf("could not set params: %w", err)
		}

		if err := k.SetAutomations(innerCtx, genState.Automations); err != nil {
			return fmt.Errorf("could not set automations: %w", err)
		}

		if err := k.SetGenesisAutomationFunds(innerCtx, genState.AutomationFunds); err != nil {
			return fmt.Errorf("could not set automation funds: %w", err)
		}

		k.SetAutomationsNextIndex(innerCtx, 1)

		return nil
	}); err != nil {
		panic(err)
	}

}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = types.Params{}

	genesis.Automations = k.GetAutomations(ctx)
	genesis.AutomationFunds = k.GetAllAutomationFunds(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
