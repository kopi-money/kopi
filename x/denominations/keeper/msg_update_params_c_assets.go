package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) CAssetAddDenom(ctx context.Context, req *types.MsgCAssetAddDenom) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

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

		baseDenom, err := k.GetDexDenom(innerCtx, req.BaseDenom)
		if err != nil {
			return err
		}

		if !k.IsValidDenom(innerCtx, req.Name) {
			dexDenom, ratio, err := k.createDexDenom(innerCtx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, baseDenom.Exponent)
			if err != nil {
				return err
			}

			params.DexDenoms = append(params.DexDenoms, dexDenom)

			k.ratios.Set(innerCtx, req.Name, ratio)
		}

		return k.SetParams(innerCtx, params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) CAssetUpdateReference(ctx context.Context, req *types.MsgCAssetUpdateReference) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets
		return k.SetParams(innerCtx, params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) CAssetUpdateDexFeeShare(ctx context.Context, req *types.MsgCAssetUpdateDexFeeShare) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets
		return k.SetParams(innerCtx, params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) CAssetUpdateBorrowLimit(ctx context.Context, req *types.MsgCAssetUpdateBorrowLimit) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets
		return k.SetParams(innerCtx, params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) CAssetUpdateMinimumLoanSize(ctx context.Context, req *types.MsgCAssetUpdateMinimumLoanSize) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

		minimumLoanSize, ok := math.NewIntFromString(req.MinimumLoanSize)
		if !ok {
			return types.ErrInvalidAmount
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets
		return k.SetParams(innerCtx, params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}
