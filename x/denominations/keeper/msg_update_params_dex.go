package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/constants"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) DexAddDenom(ctx context.Context, req *types.MsgDexAddDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	dexDenom, ratio, err := k.CreateDexDenom(ctx, req.Name, req.Factor, req.MinTradeLiquidity, req.MinOrderSize, req.MinDexLiquidity, req.Exponent)
	if err != nil {
		return nil, err
	}

	if err = k.Keeper.DexAddDenom(ctx, dexDenom, ratio); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, err
}

func (k Keeper) DexAddDenom(ctx context.Context, dexDenom types.DexDenom, ratio types.Ratio) error {
	params := k.GetParams(ctx)
	params.DexDenoms = append(params.DexDenoms, dexDenom)

	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	k.ratios.Set(ctx, dexDenom.Name, ratio)

	return nil
}

func (k msgServer) DexUpdateMinimumDexLiquidity(ctx context.Context, req *types.MsgDexUpdateMinimumDexLiquidity) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.Keeper.DexUpdateMinimumDexLiquidity(ctx, req.Name, req.MinimumDexLiquidity); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) DexUpdateMinimumTradeLiquidity(ctx context.Context, req *types.MsgDexUpdateMinimumTradeLiquidity) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.Keeper.DexUpdateMinimumTradeLiquidity(ctx, req.Name, req.MinimumTradeLiquidity); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) DexUpdateMinimumTradeLiquidity(ctx context.Context, denom, minTradeLiquidityStr string) error {
	minTradeLiquidity, ok := math.NewIntFromString(minTradeLiquidityStr)
	if !ok {
		return fmt.Errorf("invalid int: %v", minTradeLiquidityStr)
	}

	return k.SetMinimumTradeLiquidity(ctx, denom, minTradeLiquidity)
}

func (k Keeper) SetMinimumTradeLiquidity(ctx context.Context, denom string, minimumTradeLiquidity math.Int) error {
	params := k.GetParams(ctx)
	dexDenoms := []types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == denom {
			dexDenom.MinTradeLiquidity = minimumTradeLiquidity
			found = true
		}

		dexDenoms = append(dexDenoms, dexDenom)
	}

	if !found {
		return types.ErrInvalidDexAsset
	}

	params.DexDenoms = dexDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	return nil
}

func (k Keeper) DexUpdateMinimumDexLiquidity(ctx context.Context, denom, minimumDexLiquidityStr string) error {
	var minimumDexLiquidity *math.Int
	if minimumDexLiquidityStr != "" {
		mdl, ok := math.NewIntFromString(minimumDexLiquidityStr)
		if !ok {
			return fmt.Errorf("invalid minimum dex liquidity: %s", minimumDexLiquidityStr)
		}

		minimumDexLiquidity = &mdl
	}

	return k.SetMinimumDexLiquidity(ctx, denom, minimumDexLiquidity)
}

func (k Keeper) SetMinimumDexLiquidity(ctx context.Context, denom string, minimumDexLiquidity *math.Int) error {
	params := k.GetParams(ctx)

	dexDenoms := []types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == denom {
			dexDenom.MinDexLiquidity = minimumDexLiquidity
			found = true
		}

		dexDenoms = append(dexDenoms, dexDenom)
	}

	if !found {
		return types.ErrInvalidDexAsset
	}

	params.DexDenoms = dexDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	return nil
}

func (k msgServer) DexUpdateMinimumOrderSize(ctx context.Context, req *types.MsgDexUpdateMinimumOrderSize) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	minOrderSize, _ := math.NewIntFromString(req.MinOrderSize)
	dexDenoms := []types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == req.Name {
			dexDenom.MinOrderSize = minOrderSize
			found = true
		}

		dexDenoms = append(dexDenoms, dexDenom)
	}

	if !found {
		return nil, types.ErrInvalidDexAsset
	}

	params.DexDenoms = dexDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) CreateDexDenom(ctx context.Context, name, factorStr, minTradeLiquidityStr, minOrderSizeStr, minDexLiquidityStr string, exponent uint64) (types.DexDenom, types.Ratio, error) {
	ratioFactor, err := k.CreateRatio(ctx, factorStr, exponent)
	if err != nil {
		return types.DexDenom{}, types.Ratio{}, err
	}

	minTradeLiquidity, ok := math.NewIntFromString(minTradeLiquidityStr)
	if !ok {
		return types.DexDenom{}, types.Ratio{}, fmt.Errorf("invalid min trade liquidity")
	}

	minDexLiquidity, ok := math.NewIntFromString(minTradeLiquidityStr)
	if !ok {
		return types.DexDenom{}, types.Ratio{}, fmt.Errorf("invalid min dex liquidity")
	}

	minOrderSize, ok := math.NewIntFromString(minOrderSizeStr)
	if !ok {
		return types.DexDenom{}, types.Ratio{}, fmt.Errorf("invalid min order size")
	}

	dexDenom := types.DexDenom{
		Name:              name,
		MinTradeLiquidity: minTradeLiquidity,
		MinDexLiquidity:   &minDexLiquidity,
		MinOrderSize:      minOrderSize,
		Exponent:          exponent,
	}

	ratio := types.Ratio{
		Denom: name,
		Ratio: ratioFactor,
	}

	return dexDenom, ratio, nil
}

func (k Keeper) CreateRatio(ctx context.Context, factorStr string, exponent uint64) (math.LegacyDec, error) {
	referenceFactor, referenceDenom, err := types.ExtractNumberAndString(factorStr)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return k.CreateRatioFromReference(ctx, referenceFactor, referenceDenom, exponent)
}

func (k Keeper) CreateRatioFromReference(ctx context.Context, referenceFactor math.LegacyDec, referenceDenom string, exponent uint64) (math.LegacyDec, error) {
	if !referenceFactor.IsPositive() {
		return math.LegacyDec{}, types.ErrInvalidFactor
	}

	if referenceDenom != "" {
		var otherRatio types.Ratio
		otherRatio, err := k.GetRatio(ctx, referenceDenom)
		if err != nil {
			return math.LegacyDec{}, fmt.Errorf("unable to find ratio for %s: %w", referenceDenom, err)
		}

		referenceFactor = otherRatio.Ratio.Quo(referenceFactor) // C
	} else {
		referenceDenom = constants.BaseCurrency
	}

	otherDenom, err := k.GetDexDenom(ctx, referenceDenom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	referenceFactor = adjustForExponent(referenceFactor, otherDenom.Exponent, exponent)
	if !referenceFactor.IsPositive() {
		return math.LegacyDec{}, types.ErrInvalidFactor
	}

	return referenceFactor, nil
}

func adjustForExponent(value math.LegacyDec, exp1, exp2 uint64) math.LegacyDec {
	if exp1 == exp2 {
		return value
	}

	value = value.Mul(math.LegacyNewDec(10).Power(exp2))
	value = value.Quo(math.LegacyNewDec(10).Power(exp1)) // C
	return value
}
