package v21

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUpgradeHandler(_ *module.Manager, _ module.Configurator, denomK denomkeeper.Keeper, dexK dexkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		hundredK := math.NewInt(100_000_000000)
		twfiK := math.NewInt(25_000_000000)
		tenK := math.NewInt(10_000_000000)
		oneK := math.NewInt(1_000_000000)
		one := math.NewInt(1_000000)
		zero := math.ZeroInt()

		if err := cache.Transact(ctx, func(innerCtx context.Context) error {
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ukopi", hundredK)

			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ukusd", &tenK)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "uckusd", &one)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ukusd", twfiK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "uckusd", tenK)

			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ibc/295548A78785A1007F232DE286149A6FF512F180AF5657780FC89C009E2C348F", &one)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ucusdc", &one)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "uasusdc", &one)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ibc/295548A78785A1007F232DE286149A6FF512F180AF5657780FC89C009E2C348F", twfiK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ucusdc", tenK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "uasusdc", tenK)

			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ibc/D8A36AE90F20FE4843A8D249B1BCF0CCDDE35C4B605C8DED57BED20C639162D0", &zero)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ucusdtinj", &zero)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "uasusdtinj", &zero)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ibc/D8A36AE90F20FE4843A8D249B1BCF0CCDDE35C4B605C8DED57BED20C639162D0", twfiK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ucusdtinj", tenK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "uasusdtinj", tenK)

			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ibc/376222D6D9DAE23092E29740E56B758580935A6D77C24C2ABD57A6A78A1F3955", &one)
			_ = denomK.SetMinimumDexLiquidity(innerCtx, "ibc/A2E2EEC9057A4A1C2C0A6A4C78B0239118DF5F278830F50B4A6BDD7A66506B78", &one)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ibc/376222D6D9DAE23092E29740E56B758580935A6D77C24C2ABD57A6A78A1F3955", oneK)
			_ = denomK.SetMinimumTradeLiquidity(innerCtx, "ibc/A2E2EEC9057A4A1C2C0A6A4C78B0239118DF5F278830F50B4A6BDD7A66506B78", oneK)

			fac := math.LegacyNewDecWithPrec(9999, 4)
			if err := dexK.SetPriceIncreasingFactor(innerCtx, fac); err != nil {
				return fmt.Errorf("price increasing factor: %w", err)
			}

			return nil
		}); err != nil {
			return nil, fmt.Errorf("setting params: %w", err)
		}

		return vm, nil
	}
}
