package v0_3_3

import (
	"context"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateUpgradeHandler small security fix, can be a no-op, running mm.RunMigarions just to be sure
func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}
