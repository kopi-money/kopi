package app

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	appupgradetypes "github.com/kopi-money/kopi/app/upgrades"

	"github.com/kopi-money/kopi/app/upgrades/v0_6_1"
	"github.com/kopi-money/kopi/app/upgrades/v0_6_2"
	"github.com/kopi-money/kopi/app/upgrades/v0_6_3"
	"github.com/kopi-money/kopi/app/upgrades/v0_6_4"
	"github.com/kopi-money/kopi/app/upgrades/v0_6_5_1"
	"github.com/kopi-money/kopi/app/upgrades/v0_6_5_2"
	v11 "github.com/kopi-money/kopi/app/upgrades/v11"
	v12 "github.com/kopi-money/kopi/app/upgrades/v12"
	v13 "github.com/kopi-money/kopi/app/upgrades/v13"
	v14 "github.com/kopi-money/kopi/app/upgrades/v14"
	v15 "github.com/kopi-money/kopi/app/upgrades/v15"
	v16 "github.com/kopi-money/kopi/app/upgrades/v16"
	v17 "github.com/kopi-money/kopi/app/upgrades/v17"
	v18 "github.com/kopi-money/kopi/app/upgrades/v18"
	v19 "github.com/kopi-money/kopi/app/upgrades/v19"
	v20 "github.com/kopi-money/kopi/app/upgrades/v20"
	v21 "github.com/kopi-money/kopi/app/upgrades/v21"
	v22 "github.com/kopi-money/kopi/app/upgrades/v22"
	v7 "github.com/kopi-money/kopi/app/upgrades/v7"
	v8 "github.com/kopi-money/kopi/app/upgrades/v8"
	v9 "github.com/kopi-money/kopi/app/upgrades/v9"
)

func (app *App) setupUpgradeHandlers(appOpts servertypes.AppOptions) error {
	upgrades := []appupgradetypes.Upgrade{
		{
			UpgradeName:          v0_6_1.UpgradeName,
			CreateUpgradeHandler: v0_6_1.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v0_6_2.UpgradeName,
			CreateUpgradeHandler: v0_6_2.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v0_6_3.UpgradeName,
			CreateUpgradeHandler: v0_6_3.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v0_6_4.UpgradeName,
			CreateUpgradeHandler: v0_6_4.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v0_6_5_1.UpgradeName,
			CreateUpgradeHandler: v0_6_5_1.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v0_6_5_2.UpgradeName,
			CreateUpgradeHandler: v0_6_5_2.CreateUpgradeHandler,
		},
		{
			UpgradeName: v7.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v7.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper, app.ReserveKeeper, app.WasmKeeper)
			},
			StoreUpgrades: storetypes.StoreUpgrades{
				Added: []string{wasmtypes.ModuleName},
			},
		},
		{
			UpgradeName: v7.UpgradeName_rc3,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v7.CreateUpgradeHandler_rc3(manager, configurator, app.DenominationsKeeper, app.DexKeeper, app.ReserveKeeper, app.WasmKeeper)
			},
		},
		{
			UpgradeName: v7.UpgradeName_rc5,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v7.CreateUpgradeHandler_rc5(manager, configurator)
			},
		},
		{
			UpgradeName: v7.UpgradeName_rc6,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v7.CreateUpgradeHandler_rc6(manager, configurator)
			},
		},
		{
			UpgradeName: v8.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v8.CreateUpgradeHandler(manager, configurator)
			},
		},
		{
			UpgradeName: v9.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v9.CreateUpgradeHandler(manager, configurator)
			},
		},
		{
			UpgradeName:          v11.UpgradeName_rc1,
			CreateUpgradeHandler: v11.CreateUpgradeHandler_rc1,
		},
		{
			UpgradeName:          v11.UpgradeName_rc2,
			CreateUpgradeHandler: v11.CreateUpgradeHandler_rc2,
		},
		{
			UpgradeName: v11.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v11.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName:          v12.UpgradeName_rc1,
			CreateUpgradeHandler: v12.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v12.UpgradeName_rc2,
			CreateUpgradeHandler: v12.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v12.UpgradeName_rc3,
			CreateUpgradeHandler: v12.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v12.UpgradeName,
			CreateUpgradeHandler: v12.CreateUpgradeHandler,
		},
		{
			UpgradeName: v13.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v13.CreateUpgradeHandler(manager, configurator, app.IBCKeeper)
			},
		},
		{
			UpgradeName:          v14.UpgradeName,
			CreateUpgradeHandler: v14.CreateUpgradeHandler,
		},
		{
			UpgradeName: v15.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v15.CreateUpgradeHandler(manager, configurator, app.IBCKeeper)
			},
		},
		{
			UpgradeName: v16.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v16.CreateUpgradeHandler(manager, configurator, app.IBCKeeper)
			},
		},
		{
			UpgradeName:          v17.UpgradeName,
			CreateUpgradeHandler: v17.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v18.UpgradeName,
			CreateUpgradeHandler: v18.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeName,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeNameRC1,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeNameRC3,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeNameRC11,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeNameRC12,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName:          v19.UpgradeNameRC13,
			CreateUpgradeHandler: v19.CreateUpgradeHandler,
		},
		{
			UpgradeName: v20.UpgradeNameRC1,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v20.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper)
			},
		},
		{
			UpgradeName: v20.UpgradeNameRC2,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v20.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper)
			},
		},
		{
			UpgradeName: v20.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v20.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper)
			},
		},
		{
			UpgradeName: v21.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v21.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName: v21.UpgradeNameRC1,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v21.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName: v21.UpgradeNameRC2,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v21.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName: v21.UpgradeNameRC3,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v21.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName: v22.UpgradeNameRC1,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v21.CreateUpgradeHandler(manager, configurator, app.DenominationsKeeper, app.DexKeeper)
			},
		},
		{
			UpgradeName: v22.UpgradeNameRC2,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v22.CreateUpgradeHandler(manager, configurator, app.TokenfactoryKeeper)
			},
		},
		{
			UpgradeName: v22.UpgradeNameRC7,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v22.CreateUpgradeHandler(manager, configurator, app.TokenfactoryKeeper)
			},
		},
		{
			UpgradeName: v22.UpgradeName,
			CreateUpgradeHandler: func(manager *module.Manager, configurator module.Configurator) types.UpgradeHandler {
				return v22.CreateUpgradeHandler(manager, configurator, app.TokenfactoryKeeper)
			},
		},
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return fmt.Errorf("failed to read upgrade info from disk %w", err)
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return nil
	}

	for _, upgrade := range upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(app.ModuleManager, app.Configurator()),
		)

		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades))
		}
	}

	return nil
}
