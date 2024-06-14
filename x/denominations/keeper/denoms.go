package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/denominations/types"
)

// IsValidDenom is used to check whether a given denom is included in the parameters
func (k Keeper) IsValidDenom(ctx context.Context, denom string) bool {
	return contains(k.Denoms(ctx), denom)
}

// Denoms returns a list of all denoms
func (k Keeper) Denoms(ctx context.Context) (denoms []string) {
	for _, dexDenom := range k.GetParams(ctx).DexDenoms {
		denoms = append(denoms, dexDenom.Name)
	}

	return
}

func (k Keeper) IsKCoin(ctx context.Context, denom string) bool {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		if kCoin.Denom == denom {
			return true
		}
	}

	return false
}

func (k Keeper) IsNativeDenom(ctx context.Context, denom string) bool {
	return denom == utils.BaseCurrency || k.IsKCoin(ctx, denom)
}

// KCoins returns a slice containing the kCoins of all denom groups.
func (k Keeper) KCoins(ctx context.Context) (kCoins []string) {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		kCoins = append(kCoins, kCoin.Denom)
	}

	return kCoins
}

// NonKCoins returns a slice containing the non-kCoins of all denom groups.
func (k Keeper) NonKCoins(ctx context.Context) (nonKCoins []string) {
	for _, dexDenom := range k.GetParams(ctx).DexDenoms {
		if !k.IsValidDenom(ctx, dexDenom.Name) {
			nonKCoins = append(nonKCoins, dexDenom.Name)
		}
	}

	return
}

// ReferenceDenoms returns a list of denoms that are used as price reference for a kCoin. If the kCoin
// does not exist, an empty slice is created.
func (k Keeper) ReferenceDenoms(ctx context.Context, kCoinName string) []string {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		if kCoin.Denom == kCoinName {
			return kCoin.References
		}
	}

	return nil
}

// InitialVirtualLiquidityFactor returns the factor used for initial virtual liquidity for a denom.
func (k Keeper) InitialVirtualLiquidityFactor(ctx context.Context, denom string) (math.LegacyDec, error) {
	for _, dexDenom := range k.GetParams(ctx).DexDenoms {
		if dexDenom.Name == denom {
			return *dexDenom.Factor, nil
		}
	}

	return math.LegacyDec{}, fmt.Errorf("no initial virtual liquidity factor found for %v", denom)
}

func (k Keeper) MaxSupply(ctx context.Context, kCoinName string) math.Int {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		if kCoin.Denom == kCoinName {
			return kCoin.MaxSupply
		}
	}

	panic(fmt.Sprintf("no max burn amount found for %v", kCoinName))
}

func (k Keeper) MaxBurnAmount(ctx context.Context, kCoinName string) math.Int {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		if kCoin.Denom == kCoinName {
			return kCoin.MaxBurnAmount
		}
	}

	panic(fmt.Sprintf("no max burn amount found for %v", kCoinName))
}

func (k Keeper) MaxMintAmount(ctx context.Context, kCoinName string) math.Int {
	for _, kCoin := range k.GetParams(ctx).KCoins {
		if kCoin.Denom == kCoinName {
			return kCoin.MaxMintAmount
		}
	}

	panic(fmt.Sprintf("no max mint amount found for %v", kCoinName))
}

func (k Keeper) MinLiquidity(ctx context.Context, denom string) math.Int {
	for _, dexDenom := range k.GetParams(ctx).DexDenoms {
		if dexDenom.Name == denom {
			return dexDenom.MinLiquidity
		}
	}

	panic(fmt.Sprintf("no minimum liquidity found for %v", denom))
}

func (k Keeper) MinOrderSize(ctx context.Context, denom string) math.Int {
	for _, dexDenom := range k.GetParams(ctx).DexDenoms {
		if dexDenom.Name == denom {
			return dexDenom.MinOrderSize
		}
	}

	panic(fmt.Sprintf("no minimum order size found for %v", denom))
}

func (k Keeper) GetCAssets(ctx context.Context) []*types.CAsset {
	return k.GetParams(ctx).CAssets
}

func (k Keeper) GetCAssetByBaseName(ctx context.Context, baseDenom string) (*types.CAsset, error) {
	for _, aasset := range k.GetParams(ctx).CAssets {
		if aasset.BaseDenom == baseDenom {
			return aasset, nil
		}
	}

	return nil, types.ErrInvalidCAsset
}

func (k Keeper) GetCAssetByName(ctx context.Context, name string) (*types.CAsset, error) {
	for _, aasset := range k.GetParams(ctx).CAssets {
		if aasset.Name == name {
			return aasset, nil
		}
	}

	return nil, types.ErrInvalidCAsset
}

func (k Keeper) IsBorrowableDenom(ctx context.Context, denom string) bool {
	for _, aasset := range k.GetParams(ctx).CAssets {
		if aasset.BaseDenom == denom {
			return true
		}
	}

	return false
}

func (k Keeper) GetCollateralDenoms(ctx context.Context) []*types.CollateralDenom {
	return k.GetParams(ctx).CollateralDenoms
}

func (k Keeper) GetCollateralDenom(ctx context.Context, denom string) *types.CollateralDenom {
	for _, collateralDenom := range k.GetParams(ctx).CollateralDenoms {
		if collateralDenom.Denom == denom {
			return collateralDenom
		}
	}

	return nil
}

func (k Keeper) GetDepositCap(ctx context.Context, denom string) (math.Int, error) {
	for _, collateralDenom := range k.GetParams(ctx).CollateralDenoms {
		if collateralDenom.Denom == denom {
			return collateralDenom.MaxDeposit, nil
		}
	}

	return math.Int{}, types.ErrInvalidCollateralDenom
}

func (k Keeper) GetLTV(ctx context.Context, denom string) (math.LegacyDec, error) {
	for _, collateralDenom := range k.GetParams(ctx).CollateralDenoms {
		if collateralDenom.Denom == denom {
			return collateralDenom.Ltv, nil
		}
	}

	return math.LegacyDec{}, types.ErrInvalidCollateralDenom
}

func (k Keeper) IsValidCollateralDenom(ctx context.Context, denom string) bool {
	for _, depositDenom := range k.GetParams(ctx).CollateralDenoms {
		if depositDenom.Denom == denom {
			return true
		}
	}

	return false
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}

	return false
}
