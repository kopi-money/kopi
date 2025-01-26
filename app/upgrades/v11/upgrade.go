package v11

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUpgradeHandler_rc1(_ *module.Manager, _ module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}

func CreateUpgradeHandler_rc2(_ *module.Manager, _ module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}
