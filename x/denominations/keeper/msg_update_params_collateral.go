package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddCollateralDenom(goCtx context.Context, req *types.MsgAddCollateralDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	ltv, err := math.LegacyNewDecFromStr(req.Ltv)
	if err != nil {
		return nil, err
	}

	maxDeposit, ok := math.NewIntFromString(req.MaxDeposit)
	if !ok {
		return nil, fmt.Errorf("invalid max deposit value: %v", req.MaxDeposit)
	}

	collateralDenom := types.CollateralDenom{
		Denom:      req.Denom,
		Ltv:        ltv,
		MaxDeposit: maxDeposit,
	}

	params.CollateralDenoms = append(params.CollateralDenoms, &collateralDenom)

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateCollateralDenomLTV(goCtx context.Context, req *types.MsgUpdateCollateralDenomLTV) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	ltv, err := math.LegacyNewDecFromStr(req.Ltv)
	if err != nil {
		return nil, err
	}

	collateralDenoms := []*types.CollateralDenom{}
	found := false

	for _, collateralDenom := range params.CollateralDenoms {
		if collateralDenom.Denom == req.Denom {
			collateralDenom.Ltv = ltv
			found = true
		}

		collateralDenoms = append(collateralDenoms, collateralDenom)
	}

	if !found {
		return nil, types.ErrInvalidCollateralDenom
	}

	params.CollateralDenoms = collateralDenoms

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateCollateralDenomMaxDeposit(goCtx context.Context, req *types.MsgUpdateCollateralDenomMaxDeposit) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	maxDeposit, ok := math.NewIntFromString(req.MaxDeposit)
	if !ok {
		return nil, fmt.Errorf("invalid max deposit value: %v", req.MaxDeposit)
	}

	collateralDenoms := []*types.CollateralDenom{}
	found := false

	for _, collateralDenom := range params.CollateralDenoms {
		if collateralDenom.Denom == req.Denom {
			collateralDenom.MaxDeposit = maxDeposit
			found = true
		}

		collateralDenoms = append(collateralDenoms, collateralDenom)
	}

	if !found {
		return nil, types.ErrInvalidCollateralDenom
	}

	params.CollateralDenoms = collateralDenoms

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
