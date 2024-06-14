package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"github.com/kopi-money/kopi/cache"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddCAsset(goCtx context.Context, req *types.MsgAddCAsset) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		dexFeeShare, err := math.LegacyNewDecFromStr(req.DexFeeShare)
		if err != nil {
			return err
		}

		borrowLimit, err := math.LegacyNewDecFromStr(req.BorrowLimit)
		if err != nil {
			return err
		}

		factor, err := math.LegacyNewDecFromStr(req.Factor)
		if err != nil {
			return err
		}

		minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
		if !ok {
			return fmt.Errorf("given min liquidity is no valid math.Int: \"%v\"", req.MinLiquidity)
		}

		minOrderSize, ok := math.NewIntFromString(req.MinOrderSize)
		if !ok {
			return fmt.Errorf("given min order size is no valid math.Int: \"%v\"", req.MinOrderSize)
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
				MinOrderSize: minOrderSize,
			})
		}

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateCAssetDexFeeShare(goCtx context.Context, req *types.MsgUpdateCAssetDexFeeShare) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		dexFeeShare, err := math.LegacyNewDecFromStr(req.DexFeeShare)
		if err != nil {
			return err
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateCAssetBorrowLimit(goCtx context.Context, req *types.MsgUpdateCAssetBorrowLimit) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		borrowLimit, err := math.LegacyNewDecFromStr(req.BorrowLimit)
		if err != nil {
			return err
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateCAssetMinimumLoanSize(goCtx context.Context, req *types.MsgUpdateCAssetMinimumLoanSize) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		minimumLoanSize, ok := math.NewIntFromString(req.MinimumLoanSize)
		if !ok {
			return types.ErrInvalidAmount
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
			return types.ErrInvalidCAsset
		}

		params.CAssets = cAssets

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}
