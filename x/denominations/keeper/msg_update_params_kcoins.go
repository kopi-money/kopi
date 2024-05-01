package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddKCoin(goCtx context.Context, req *types.MsgAddKCoin) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	maxSupply, ok := math.NewIntFromString(req.MaxSupply)
	if !ok {
		return nil, fmt.Errorf("invalid max supply value: %v", req.MaxSupply)
	}

	maxBurnAmount, ok := math.NewIntFromString(req.MaxBurnAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max burn amount value: %v", req.MaxBurnAmount)
	}

	maxMintAmount, ok := math.NewIntFromString(req.MaxMintAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max mint amount value: %v", req.MaxMintAmount)
	}

	factor, err := math.LegacyNewDecFromStr(req.Factor)
	if err != nil {
		return nil, err
	}

	minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
	if !ok {
		return nil, fmt.Errorf("invalid min liquidity value: %v", req.MinLiquidity)
	}

	kCoin := types.KCoin{
		Denom:         req.Denom,
		References:    req.References,
		MaxSupply:     maxSupply,
		MaxMintAmount: maxMintAmount,
		MaxBurnAmount: maxBurnAmount,
	}

	dexDenom := types.DexDenom{
		Name:         req.Denom,
		Factor:       &factor,
		MinLiquidity: minLiquidity,
	}

	params.KCoins = append(params.KCoins, &kCoin)
	params.DexDenoms = append(params.DexDenoms, &dexDenom)

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateKCoinSupply(goCtx context.Context, req *types.MsgUpdateKCoinSupply) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	maxSupply, ok := math.NewIntFromString(req.MaxSupply)
	if !ok {
		return nil, fmt.Errorf("invalid max supply value: %v", req.MaxSupply)
	}

	kCoins := []*types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.Denom == req.Denom {
			kCoin.MaxSupply = maxSupply
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateKCoinMintAmount(goCtx context.Context, req *types.MsgUpdateKCoinMintAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	maxMintAmount, ok := math.NewIntFromString(req.MaxMintAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max mint amount value: %v", req.MaxMintAmount)
	}

	kCoins := []*types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.Denom == req.Denom {
			kCoin.MaxMintAmount = maxMintAmount
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateKCoinBurnAmount(goCtx context.Context, req *types.MsgUpdateKCoinBurnAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	maxBurnAmount, ok := math.NewIntFromString(req.MaxBurnAmount)
	if !ok {
		return nil, fmt.Errorf("invalid max burn amount value: %v", req.MaxBurnAmount)
	}

	kCoins := []*types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.Denom == req.Denom {
			kCoin.MaxBurnAmount = maxBurnAmount
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) AddKCoinReferences(goCtx context.Context, req *types.MsgAddKCoinReferences) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	kCoins := []*types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.Denom == req.Denom {
			kCoin.References = append(kCoin.References, req.References...)
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) RemoveKCoinReferences(goCtx context.Context, req *types.MsgRemoveKCoinReferences) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	kCoins := []*types.KCoin{}
	found := false

	for _, kCoin := range params.KCoins {
		if kCoin.Denom == req.Denom {
			kCoin.References = filterReferences(kCoin.References, req.References)
			found = true
		}

		kCoins = append(kCoins, kCoin)
	}

	if !found {
		return nil, types.ErrInvalidKCoin
	}

	params.KCoins = kCoins

	if err := params.Validate(); err != nil {
		return nil, err
	}

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
