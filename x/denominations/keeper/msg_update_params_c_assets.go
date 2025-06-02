package keeper

import (
	"context"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) CAssetAddDenom(ctx context.Context, req *types.MsgCAssetAddDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	dexFeeShare, _ := math.LegacyNewDecFromStr(req.DexFeeShare)
	borrowLimit, _ := math.LegacyNewDecFromStr(req.BorrowLimit)
	minimumLoanSize, _ := math.NewIntFromString(req.MinLoanSize)

	params.CAssets = append(params.CAssets, types.CAsset{
		DexDenom:        req.Name,
		BaseDexDenom:    req.BaseDenom,
		DexFeeShare:     dexFeeShare,
		BorrowLimit:     borrowLimit,
		MinimumLoanSize: minimumLoanSize,
	})

	baseDenom, err := k.GetDexDenom(ctx, req.BaseDenom)
	if err != nil {
		return nil, err
	}

	if !k.IsValidDenom(ctx, req.Name) {
		dexDenom, ratio, err := k.CreateDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, req.MinVirtualLiquidity, baseDenom.Exponent)
		if err != nil {
			return nil, err
		}

		params.DexDenoms = append(params.DexDenoms, dexDenom)

		k.ratios.Set(ctx, req.Name, ratio)
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) CAssetUpdateReference(ctx context.Context, req *types.MsgCAssetUpdateReference) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	var (
		cAssets []types.CAsset
		found   bool
	)

	for _, cAsset := range params.CAssets {
		if cAsset.DexDenom == req.Name {
			cAsset.BaseDexDenom = req.NewBaseDenom
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) CAssetUpdateDexFeeShare(ctx context.Context, req *types.MsgCAssetUpdateDexFeeShare) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	dexFeeShare, _ := math.LegacyNewDecFromStr(req.DexFeeShare)

	var (
		cAssets []types.CAsset
		found   bool
	)

	for _, cAsset := range params.CAssets {
		if cAsset.DexDenom == req.Name {
			cAsset.DexFeeShare = dexFeeShare
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) CAssetUpdateBorrowLimit(ctx context.Context, req *types.MsgCAssetUpdateBorrowLimit) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	borrowLimit, _ := math.LegacyNewDecFromStr(req.BorrowLimit)

	var (
		cAssets []types.CAsset
		found   bool
	)

	for _, cAsset := range params.CAssets {
		if cAsset.DexDenom == req.Name {
			cAsset.BorrowLimit = borrowLimit
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) CAssetUpdateMinimumLoanSize(ctx context.Context, req *types.MsgCAssetUpdateMinimumLoanSize) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	minimumLoanSize, ok := math.NewIntFromString(req.MinimumLoanSize)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	var (
		cAssets []types.CAsset
		found   bool
	)

	for _, cAsset := range params.CAssets {
		if cAsset.DexDenom == req.Name {
			cAsset.MinimumLoanSize = minimumLoanSize
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
