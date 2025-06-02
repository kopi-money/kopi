package v15

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, ibcK *ibckeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		if channel, has := ibcK.ChannelKeeper.GetChannel(sdkCtx, "transfer", "channel-1"); has {
			channel.ConnectionHops = []string{"connection-41"}
			ibcK.ChannelKeeper.SetChannel(sdkCtx, "transfer", "channel-1", channel)
		}

		return vm, nil
	}
}
