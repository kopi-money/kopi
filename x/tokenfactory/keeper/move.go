package keeper

import (
	"context"
	"fmt"
	gomath "math"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) MoveDenom(ctx context.Context, factoryDenom types.FactoryDenom) error {
	if factoryDenom.Moved {
		return types.ErrDenomAlreadyMoved
	}

	pool, has := k.GetLiquidityPool(ctx, factoryDenom.FullName)
	if !has {
		return types.ErrPoolDoesNotExist
	}

	if pool.ThresholdCrossed == nil {
		return types.ErrPoolTresholdNotCrossed
	}

	blocktime := sdk.UnwrapSDKContext(ctx).BlockTime()
	blocktime = blocktime.Add(time.Duration(k.poolThresholdSeconds(ctx)) * time.Second)

	if blocktime.After(*pool.ThresholdCrossed) {
		return types.ErrPoolTresholdCrossedTooRecently
	}

	poolRatio, err := pool.GetPoolRatio()
	if err != nil {
		return fmt.Errorf("pool ratio: %w", err)
	}

	ratioFactor, err := k.DenomKeeper.CreateRatioFromReference(ctx, poolRatio, factoryDenom.LocalName, pool.KCoin, factoryDenom.Exponent)
	if err != nil {
		return fmt.Errorf("create ratio from reference: %w", err)
	}

	oneUnit, err := oneBaseUnit(factoryDenom.Exponent)
	if err != nil {
		return fmt.Errorf("get one unit: %w", err)
	}

	dexDenom := denomtypes.DexDenom{
		Name:              factoryDenom.LocalName,
		Exponent:          factoryDenom.Exponent,
		MinDexLiquidity:   &oneUnit,
		MinTradeLiquidity: oneUnit,
		MinOrderSize:      oneUnit,
	}

	ratio := denomtypes.Ratio{
		Denom: factoryDenom.LocalName,
		Ratio: ratioFactor,
	}

	if err = k.DenomKeeper.DexAddDenom(ctx, dexDenom, ratio); err != nil {
		return fmt.Errorf("dex add denom: %w", err)
	}

	if err = k.moveLiquidity(ctx, factoryDenom, pool); err != nil {
		return fmt.Errorf("move liquidity: %w", err)
	}

	factoryDenom.Moved = true
	k.SetDenom(ctx, factoryDenom)

	return nil
}

func (k Keeper) moveLiquidity(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool) error {
	ratio, err := pool.GetPoolRatio()
	if err != nil {
		return err
	}

	shareIterator := k.LiquidityShareIterator(ctx, factoryDenom.FullName)

	var denom string
	if factoryDenom.LocalName != "" {
		denom = factoryDenom.LocalName
	} else {
		denom = factoryDenom.FullName
	}

	for shareIterator.Valid() {
		keyValue := shareIterator.GetNextKeyValue()

		amountFactory := pool.FactoryDenomAmount.ToLegacyDec().Mul(keyValue.Value().Value().Share)
		amountKCoin := amountFactory.Mul(ratio)

		coins := sdk.NewCoins(
			sdk.NewCoin(denom, amountFactory.TruncateInt()),
			sdk.NewCoin(pool.KCoin, amountKCoin.TruncateInt()),
		)

		acc, _ := sdk.AccAddressFromBech32(keyValue.Key())
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
			return fmt.Errorf("send coins from module to account: %w", err)
		}

		if err = k.DexKeeper.AddLiquidityWithCompound(ctx, acc, denom, amountFactory.TruncateInt(), true); err != nil {
			return fmt.Errorf("add ibc liquidity: %w", err)
		}

		if err = k.DexKeeper.AddLiquidityWithCompound(ctx, acc, pool.KCoin, amountKCoin.TruncateInt(), true); err != nil {
			return fmt.Errorf("add kcoin liquidity: %w", err)
		}
	}

	k.liquidityPools.Remove(ctx, factoryDenom.FullName)

	return nil
}

func oneBaseUnit(exponent uint64) (math.Int, error) {
	if exponent > 19 {
		return math.Int{}, fmt.Errorf("exponent must not be larger than 18")
	}
	microUnits := int64(gomath.Pow(10, float64(exponent)))
	return math.NewInt(microUnits), nil
}
