package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

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
	})

	params.DexDenoms = append(params.DexDenoms, &types.DexDenom{
		Name:         req.Name,
		Factor:       &factor,
		MinLiquidity: minLiquidity,
	})

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
