package v13

import (
	"context"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, ibcK *ibckeeper.Keeper, capabilityKeeper *capabilitykeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		channels := ibcK.ChannelKeeper.GetAllChannels(sdkCtx)

		latestIndex := capabilityKeeper.GetLatestIndex(sdkCtx)
		if latestIndex == 0 {
			if err := capabilityKeeper.InitializeIndex(sdkCtx, uint64(len(channels))); err != nil {
				return nil, err
			}
		}

		return vm, nil
	}
}
