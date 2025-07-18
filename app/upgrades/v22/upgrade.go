package v22

import (
	"context"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	factorykeeper "github.com/kopi-money/kopi/x/tokenfactory/keeper"
	factorytypes "github.com/kopi-money/kopi/x/tokenfactory/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, factoryK factorykeeper.Keeper, denomK denomkeeper.Keeper, ibcK *ibckeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if sdkCtx := sdk.UnwrapSDKContext(ctx); sdkCtx.ChainID() == "luwak-1" {
			if channel, has := ibcK.ChannelKeeper.GetChannel(sdkCtx, "transfer", "channel-1"); has {
				channel.ConnectionHops = []string{"connection-41"}
				ibcK.ChannelKeeper.SetChannel(sdkCtx, "transfer", "channel-1", channel)
			}

			if err := cache.Transact(ctx, func(innerCtx context.Context) error {
				params := denomK.GetParams(innerCtx)

				params.FactoryPoolDenoms = []denomtypes.FactoryPoolDenom{
					{
						Denom:           "ukusd",
						MinimumPoolSize: math.NewInt(100_000000),
						MoveThreshold:   math.NewInt(10_000_000000),
					},
					{
						Denom:           "ibc/295548A78785A1007F232DE286149A6FF512F180AF5657780FC89C009E2C348F",
						MinimumPoolSize: math.NewInt(100_000000),
						MoveThreshold:   math.NewInt(10_000_000000),
					},
				}

				params.FeeDenoms = []denomtypes.FeeDenom{
					{
						Denom:              "ukusd",
						MinimumTradeAmount: math.NewInt(1_000000),
					},
					{
						Denom:              "ibc/295548A78785A1007F232DE286149A6FF512F180AF5657780FC89C009E2C348F",
						MinimumTradeAmount: math.NewInt(1_000000),
					},
				}

				return denomK.SetParams(innerCtx, params)
			}); err != nil {
				return nil, err
			}

			if err := cache.Transact(ctx, func(innerCtx context.Context) error {
				return factoryK.SetParams(innerCtx, factorytypes.DefaultParams())
			}); err != nil {
				return nil, err
			}
		}

		return vm, nil
	}
}
