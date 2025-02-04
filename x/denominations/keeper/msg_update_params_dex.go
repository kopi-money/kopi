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

	params := k.GetParams(ctx)

	dexDenom, ratio, err := k.createDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, req.Exponent)
	if err != nil {
		return nil, err
	}

	params.DexDenoms = append(params.DexDenoms, dexDenom)

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	k.ratios.Set(ctx, req.Name, ratio)

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) DexUpdateMinimumLiquidity(ctx context.Context, req *types.MsgDexUpdateMinimumLiquidity) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.Keeper.DexUpdateMinimumLiquidity(ctx, req.Name, req.MinLiquidity); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) DexUpdateMinimumLiquidity(ctx context.Context, denom, minLiquidityStr string) error {
	params := k.GetParams(ctx)
	minLiquidity, _ := math.NewIntFromString(minLiquidityStr)
	dexDenoms := []types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == denom {
			dexDenom.MinLiquidity = minLiquidity
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

func (k Keeper) createDexDenom(ctx context.Context, name, factorStr, minLiquidityStr, minOrderSizeStr string, exponent uint64) (types.DexDenom, types.Ratio, error) {
	ratioFactor, err := k.CreateRatio(ctx, factorStr, exponent)
	if err != nil {
		return types.DexDenom{}, types.Ratio{}, err
	}

	minLiquidity, _ := math.NewIntFromString(minLiquidityStr)
	minOrderSize, _ := math.NewIntFromString(minOrderSizeStr)

	dexDenom := types.DexDenom{
		Name:         name,
		MinLiquidity: minLiquidity,
		MinOrderSize: minOrderSize,
		Exponent:     exponent,
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

	if !referenceFactor.IsPositive() {
		return math.LegacyDec{}, types.ErrInvalidFactor
	}

	if referenceDenom != constants.BaseCurrency && referenceDenom != "" {
		var otherRatio types.Ratio
		otherRatio, err = k.GetRatio(ctx, referenceDenom)
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
