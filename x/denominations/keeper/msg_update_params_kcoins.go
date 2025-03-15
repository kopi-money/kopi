package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) KCoinAddDenom(ctx context.Context, req *types.MsgKCoinAddDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	maxSupply, ok := math.NewIntFromString(req.MaxSupply)
	if !ok {
		return nil, fmt.Errorf("invalid max supply")
	}

	maxBurnAmount, ok := math.NewIntFromString(req.MaxBurnAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max burn amount")
	}

	maxMintAmount, ok := math.NewIntFromString(req.MaxMintAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max mint amount")
	}

	params.KCoins = append(params.KCoins, types.KCoin{
		DexDenom:      req.Name,
		References:    req.References,
		MaxSupply:     maxSupply,
		MaxMintAmount: maxMintAmount,
		MaxBurnAmount: maxBurnAmount,
	})

	dexDenom, ratio, err := k.CreateDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, req.MinVirtualLiquidity, req.Exponent)
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

func (k msgServer) KCoinUpdateSupplyLimit(ctx context.Context, req *types.MsgKCoinUpdateSupplyLimit) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	maxSupply, _ := math.NewIntFromString(req.MaxSupply)
	kCoins := []types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.DexDenom == req.Denom {
			kCoin.MaxSupply = maxSupply
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) KCoinUpdateMintAmount(ctx context.Context, req *types.MsgKCoinUpdateMintAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	maxMintAmount, _ := math.NewIntFromString(req.MaxMintAmount)
	kCoins := []types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.DexDenom == req.Denom {
			kCoin.MaxMintAmount = maxMintAmount
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) KCoinUpdateBurnAmount(ctx context.Context, req *types.MsgKCoinUpdateBurnAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	maxBurnAmount, _ := math.NewIntFromString(req.MaxBurnAmount)
	kCoins := []types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.DexDenom == req.Denom {
			kCoin.MaxBurnAmount = maxBurnAmount
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) KCoinAddReferences(ctx context.Context, req *types.MsgKCoinAddReferences) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	kCoins := []types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.DexDenom == req.Denom {
			kCoin.References = append(kCoin.References, req.References...)
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) KCoinRemoveReferences(ctx context.Context, req *types.MsgKCoinRemoveReferences) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	kCoins := []types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.DexDenom == req.Denom {
			kCoin.References = filterReferences(kCoin.References, req.References)
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func filterReferences(existingReferences, toRemove []string) (filtered []string) {
	for _, existingReference := range existingReferences {
		if !contains(toRemove, existingReference) {
			filtered = append(filtered, existingReference)
		}
	}

	return
}
