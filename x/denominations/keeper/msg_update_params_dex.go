package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/constants"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) DexAddDenom(ctx context.Context, msg *types.MsgDexAddDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	dexDenom, ratio, err := k.CreateDexDenom(ctx, msg.Name, msg.Factor, msg.MinTradeLiquidity, msg.MinOrderSize, msg.MinDexLiquidity, msg.Exponent)
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

func (k msgServer) DexUpdateMinimumDexLiquidity(ctx context.Context, msg *types.MsgDexUpdateMinimumDexLiquidity) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := k.Keeper.DexUpdateMinimumDexLiquidity(ctx, msg.Name, msg.MinimumDexLiquidity); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) DexUpdateMinimumTradeLiquidity(ctx context.Context, msg *types.MsgDexUpdateMinimumTradeLiquidity) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := k.Keeper.DexUpdateMinimumTradeLiquidity(ctx, msg.Name, msg.MinimumTradeLiquidity); err != nil {
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

func (k msgServer) DexUpdateMinimumOrderSize(ctx context.Context, msg *types.MsgDexUpdateMinimumOrderSize) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	minOrderSize, _ := math.NewIntFromString(msg.MinOrderSize)
	dexDenoms := []types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == msg.Name {
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

func (k msgServer) DexSetRatio(ctx context.Context, msg *types.MsgDexSetRatio) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if msg.Exponent < 1 {
		return nil, fmt.Errorf("exponent has to be positive, was: %v", msg.Exponent)
	}

	newRatio, err := k.CreateRatio(ctx, msg.Denom, msg.Ratio, msg.Exponent)
	if err != nil {
		return nil, fmt.Errorf("invalid ratio: %v", msg.Ratio)
	}

	if !newRatio.IsPositive() {
		return nil, fmt.Errorf("new ratio must be positive, was: %v", msg.Ratio)
	}

	ratio, err := k.GetRatio(ctx, msg.Denom)
	if err != nil {
		return nil, fmt.Errorf("unable to find ratio for %s", msg.Denom)
	}

	ratio.Ratio = newRatio
	k.SetRatio(ctx, ratio)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) CreateDexDenom(ctx context.Context, name, factorStr, minTradeLiquidityStr, minOrderSizeStr, minDexLiquidityStr string, exponent uint64) (types.DexDenom, types.Ratio, error) {
	ratioFactor, err := k.CreateRatio(ctx, name, factorStr, exponent)
	if err != nil {
		return types.DexDenom{}, types.Ratio{}, err
	}

	minTradeLiquidity, ok := math.NewIntFromString(minTradeLiquidityStr)
	if !ok {
		return types.DexDenom{}, types.Ratio{}, fmt.Errorf("invalid min trade liquidity")
	}

	minDexLiquidity, ok := math.NewIntFromString(minDexLiquidityStr)
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

func (k Keeper) CreateRatio(ctx context.Context, newDenom, factorStr string, exponent uint64) (math.LegacyDec, error) {
	referenceFactor, referenceDenom, err := types.ExtractNumberAndString(factorStr)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return k.CreateRatioFromReference(ctx, referenceFactor, newDenom, referenceDenom, exponent)
}

func (k Keeper) CreateRatioFromReference(ctx context.Context, referenceFactor math.LegacyDec, newDenom, referenceDenom string, exponent uint64) (math.LegacyDec, error) {
	if !referenceFactor.IsPositive() {
		return math.LegacyDec{}, types.ErrInvalidFactor
	}

	if referenceDenom != "" {
		otherRatio, err := k.GetRatio(ctx, referenceDenom)
		if err != nil {
			return math.LegacyDec{}, fmt.Errorf("unable to find ratio for %s: %w", referenceDenom, err)
		}

		referenceFactor = otherRatio.Ratio.Quo(referenceFactor) // C
	} else {
		referenceDenom = constants.BaseCurrency
	}

	if referenceDenom == newDenom {
		return math.LegacyDec{}, fmt.Errorf("new denom and reference denom cannot be the same")
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
