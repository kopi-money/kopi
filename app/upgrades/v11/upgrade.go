package v11

import (
	"context"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"

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

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, denomK denomkeeper.Keeper, dexK dexkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if err := cache.Transact(ctx, func(innerCtx context.Context) error {
			dexParams := dexK.GetParams(ctx)
			dexParams.TradeBaseValue = math.LegacyNewDec(1_000000_000000)
			dexParams.DiscountLevels = nil
			dexParams.PriceIncreasingFactor = math.LegacyOneDec()

			if err := dexK.SetParams(innerCtx, dexParams); err != nil {
				return err
			}

			usdRatio, _ := math.LegacyNewDecFromStr("0.25")

			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ukusd",
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "uckusd",
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5", // USDC
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ucusdc",
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "uasusdc",
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/D8A36AE90F20FE4843A8D249B1BCF0CCDDE35C4B605C8DED57BED20C639162D0", // USDT
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ucusdtinj",
				Ratio: usdRatio,
			})
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "uasusdtinj",
				Ratio: usdRatio,
			})

			osmoRatio, _ := denomK.CreateRatio(ctx, "0.35ukusd", 6)
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/646315E3B0461F5FA4C5C8968A88FC45D4D5D04A45B98F1B8294DD82F386DD85", // OSMO
				Ratio: osmoRatio,
			})

			atomRatio, _ := denomK.CreateRatio(ctx, "5.5ukusd", 6)
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/25418646C017D377ADF3202FF1E43590D0DAE3346E594E8D78176A139A928F88", // ATOM
				Ratio: atomRatio,
			})

			injRatio, _ := denomK.CreateRatio(ctx, "16ukusd", 18)
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/DE63D8AC34B752FB7D4CAA7594145EDE1C9FC256AC6D4043D0F12310EB8FC255", // INJ
				Ratio: injRatio,
			})

			lunaRatio, _ := denomK.CreateRatio(ctx, "0.29ukusd", 6)
			denomK.SetRatio(innerCtx, denomtypes.Ratio{
				Denom: "ibc/DA59C009A0B3B95E0549E6BF7B075C8239285989FF457A8EDDBB56F10B2A6986", // LUNA
				Ratio: lunaRatio,
			})

			return nil
		}); err != nil {
			return nil, err
		}

		return vm, nil
	}
}
