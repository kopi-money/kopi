package v13

import (
	"context"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, ibcK *ibckeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		clients := ibcK.ClientKeeper.GetAllClients(sdkCtx)
		ibcK.ClientKeeper.SetNextClientSequence(sdkCtx, uint64(len(clients)))

		connections := ibcK.ConnectionKeeper.GetAllConnections(sdkCtx)
		ibcK.ConnectionKeeper.SetNextConnectionSequence(sdkCtx, uint64(len(connections)))

		channels := ibcK.ChannelKeeper.GetAllChannels(sdkCtx)
		ibcK.ChannelKeeper.SetNextChannelSequence(sdkCtx, uint64(len(channels)))

		return vm, nil
	}
}
