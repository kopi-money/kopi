package v22

import (
	"context"
	"github.com/cosmos/cosmos-sdk/cache"
	factorykeeper "github.com/kopi-money/kopi/x/tokenfactory/keeper"
	factorytypes "github.com/kopi-money/kopi/x/tokenfactory/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, factoryK factorykeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if err := cache.Transact(ctx, func(innerCtx context.Context) error {
			return factoryK.SetParams(innerCtx, factorytypes.DefaultParams())
		}); err != nil {
			return nil, err
		}

		return vm, nil
	}
}
