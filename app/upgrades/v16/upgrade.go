package v16

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

		connection, has := ibcK.ConnectionKeeper.GetConnection(sdkCtx, "connection-0")
		if has {
			connection.ClientId = "07-tendermint-0"
			connection.Counterparty.ClientId = "07-tendermint-124"
			connection.Counterparty.ConnectionId = "connection-120"
			ibcK.ConnectionKeeper.SetConnection(sdkCtx, "connection-0", connection)
		}

		connection, has = ibcK.ConnectionKeeper.GetConnection(sdkCtx, "connection-41")
		if has {
			connection.Counterparty.ClientId = "07-tendermint-278"
			connection.Counterparty.ConnectionId = "connection-281"
			ibcK.ConnectionKeeper.SetConnection(sdkCtx, "connection-41", connection)
		}

		return vm, nil
	}
}
