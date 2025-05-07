package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) Init(ctx context.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return fmt.Errorf("set params: %w", err)
	}

	for _, automation := range gs.Automations {
		k.automations.Set(ctx, automation.Index, automation)
	}

	for _, automationFunds := range gs.AutomationFunds {
		k.automationFunds.Set(ctx, automationFunds.Address, types.AutomationFunds{
			Funds: automationFunds.Funds,
		})
	}

	k.automationsNextIndex.Set(ctx, gs.AutomationNextIndex)

	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := types.DefaultGenesis()

	gs.Params = k.GetParams(ctx)
	gs.Automations = k.ExportAutomations(ctx)
	gs.AutomationFunds = k.exportAutomationFunds(ctx)
	gs.ArbitrageDenoms = k.exportArbitrageDenoms(ctx)

	return gs
}
