package v20

import (
	"context"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/cosmos/cosmos-sdk/types/module"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, denomK denomkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if err := filterRatios(ctx, denomK); err != nil {
			return nil, fmt.Errorf("filter ratios: %w", err)
		}

		return vm, nil
	}
}

// filterRatios removes all ratios that are not valid denoms. Deprecated IBC denoms have been removed from the DEX but
// the function was lacking a removal of the ratio from the list, leading to some ratios still being in the list
// even though their denoms are not listed anymore.
func filterRatios(ctx context.Context, denomK denomkeeper.Keeper) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		for _, ratio := range denomK.GetAllRatios(ctx) {
			if !denomK.IsValidDenom(ctx, ratio.Denom) {
				denomK.RemoveRatio(ctx, ratio.Denom)
			}
		}

		return nil
	})
}
