package v7

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	reservekeeper "github.com/kopi-money/kopi/x/reserve/keeper"
	reservetypes "github.com/kopi-money/kopi/x/reserve/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

func CreateUpgradeHandler(mm *module.Manager, configurator module.Configurator, denomK denomkeeper.Keeper, dexK dexkeeper.Keeper, reserveK reservekeeper.Keeper, wasmK wasmkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm["capability"] = 1

		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Set CosmWasm params
		wasmParams := wasmK.GetParams(ctx)
		wasmParams.CodeUploadAccess = wasmtypes.AllowNobody
		wasmParams.InstantiateDefaultPermission = wasmtypes.AccessTypeAnyOfAddresses

		if err = wasmK.SetParams(ctx, wasmParams); err != nil {
			return vm, fmt.Errorf("unable to set CosmWasm params")
		}

		// Correct USDC ratio
		ratioUSDT, err := denomK.GetRatio(ctx, "ibc/D8A36AE90F20FE4843A8D249B1BCF0CCDDE35C4B605C8DED57BED20C639162D0")
		if err != nil {
			denomK.Logger().Info(fmt.Errorf("usdt ratio: %w", err).Error())
		} else {
			_ = cache.Transact(ctx, func(innerCtx context.Context) error {
				denomK.SetRatio(innerCtx, denomtypes.Ratio{
					Denom: "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5",
					Ratio: ratioUSDT.Ratio,
				})

				return nil
			})
		}

		// Migrate orders from V1 to V2
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			return dexK.UpgradeOrdersV2(innerCtx)
		}); err != nil {
			denomK.Logger().Info(fmt.Errorf("deleting orders v1: %w", err).Error())
		}

		// Set Reserve parameters
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			return reserveK.SetParams(innerCtx, reservetypes.Params{
				KcoinBurnShare: reservetypes.KCoinBurnShare,
				SellThreshold:  reservetypes.SellThreshold,
				BuyThreshold:   reservetypes.BuyThreshold,
			})
		}); err != nil {
			denomK.Logger().Info(fmt.Errorf("deleting orders v1: %w", err).Error())
		}

		return vm, nil
	}
}

func CreateUpgradeHandler_rc3(mm *module.Manager, configurator module.Configurator, denomK denomkeeper.Keeper, dexK dexkeeper.Keeper, reserveK reservekeeper.Keeper, wasmK wasmkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Migrate orders from V1 to V2
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			return dexK.UpgradeOrdersV2(innerCtx)
		}); err != nil {
			denomK.Logger().Info(fmt.Errorf("deleting orders v1: %w", err).Error())
		}

		// Set Reserve parameters
		if err = cache.Transact(ctx, func(innerCtx context.Context) error {
			return reserveK.SetParams(innerCtx, reservetypes.Params{
				KcoinBurnShare: reservetypes.KCoinBurnShare,
				SellThreshold:  reservetypes.SellThreshold,
				BuyThreshold:   reservetypes.BuyThreshold,
			})
		}); err != nil {
			denomK.Logger().Info(fmt.Errorf("deleting orders v1: %w", err).Error())
		}

		return vm, nil
	}
}

func CreateUpgradeHandler_rc5(_ *module.Manager, _ module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}

func CreateUpgradeHandler_rc6(_ *module.Manager, _ module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}
