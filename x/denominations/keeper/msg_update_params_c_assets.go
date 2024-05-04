package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddCAsset(goCtx context.Context, req *types.MsgAddCAsset) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	dexFeeShare, err := math.LegacyNewDecFromStr(req.DexFeeShare)
	if err != nil {
		return nil, err
	}

	borrowLimit, err := math.LegacyNewDecFromStr(req.BorrowLimit)
	if err != nil {
		return nil, err
	}

	factor, err := math.LegacyNewDecFromStr(req.Factor)
	if err != nil {
		return nil, err
	}

	minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
	if !ok {
		return nil, fmt.Errorf("given min liquidity is no valid math.Int: \"%v\"", req.MinLiquidity)
	}

	params.CAssets = append(params.CAssets, &types.CAsset{
		Name:        req.Name,
		BaseDenom:   req.BaseDenom,
		DexFeeShare: dexFeeShare,
		BorrowLimit: borrowLimit,
	})

	if !k.IsValidDenom(ctx, req.Name) {
		params.DexDenoms = append(params.DexDenoms, &types.DexDenom{
			Name:         req.Name,
			Factor:       &factor,
			MinLiquidity: minLiquidity,
		})
	}

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateCAssetDexFeeShare(goCtx context.Context, req *types.MsgUpdateCAssetDexFeeShare) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	dexFeeShare, err := math.LegacyNewDecFromStr(req.DexFeeShare)
	if err != nil {
		return nil, err
	}

	cAssets := []*types.CAsset{}
	found := false

	for _, cAsset := range params.CAssets {
		if cAsset.Name == req.Name {
			cAsset.DexFeeShare = dexFeeShare
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateCAssetBorrowLimit(goCtx context.Context, req *types.MsgUpdateCAssetBorrowLimit) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	borrowLimit, err := math.LegacyNewDecFromStr(req.BorrowLimit)
	if err != nil {
		return nil, err
	}

	cAssets := []*types.CAsset{}
	found := false

	for _, cAsset := range params.CAssets {
		if cAsset.Name == req.Name {
			cAsset.BorrowLimit = borrowLimit
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateCAssetMinimumLoanSize(goCtx context.Context, req *types.MsgUpdateCAssetMinimumLoanSize) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	minimumLoanSize, ok := math.NewIntFromString(req.MinimumLoanSize)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	cAssets := []*types.CAsset{}
	found := false

	for _, cAsset := range params.CAssets {
		if cAsset.Name == req.Name {
			cAsset.MinimumLoanSize = minimumLoanSize
			found = true
		}

		cAssets = append(cAssets, cAsset)
	}

	if !found {
		return nil, types.ErrInvalidCAsset
	}

	params.CAssets = cAssets

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
