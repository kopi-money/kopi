package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) CollateralAddDenom(ctx context.Context, req *types.MsgCollateralAddDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	ltv, _ := math.LegacyNewDecFromStr(req.Ltv)
	maxDeposit, _ := math.NewIntFromString(req.MaxDeposit)

	params.CollateralDenoms = append(params.CollateralDenoms, types.CollateralDenom{
		DexDenom:   req.Denom,
		Ltv:        ltv,
		MaxDeposit: maxDeposit,
	})

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) CollateralUpdateLTV(ctx context.Context, req *types.MsgCollateralUpdateLTV) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.Keeper.CollateralUpdateLTV(ctx, req.Denom, req.Ltv); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) CollateralUpdateLTV(ctx context.Context, denom, ltvString string) error {
	params := k.GetParams(ctx)
	ltv, err := math.LegacyNewDecFromStr(ltvString)
	if err != nil {
		return err
	}

	collateralDenoms := []types.CollateralDenom{}
	found := false

	for _, collateralDenom := range params.CollateralDenoms {
		if collateralDenom.DexDenom == denom {
			collateralDenom.Ltv = ltv
			found = true
		}

		collateralDenoms = append(collateralDenoms, collateralDenom)
	}

	if !found {
		return types.ErrInvalidCollateralDenom
	}

	params.CollateralDenoms = collateralDenoms

	return k.SetParams(ctx, params)
}

func (k msgServer) CollateralUpdateDepositLimit(ctx context.Context, req *types.MsgCollateralUpdateDepositLimit) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)

	maxDeposit, _ := math.NewIntFromString(req.MaxDeposit)
	collateralDenoms := []types.CollateralDenom{}
	found := false

	for _, collateralDenom := range params.CollateralDenoms {
		if collateralDenom.DexDenom == req.Denom {
			collateralDenom.MaxDeposit = maxDeposit
			found = true
		}

		collateralDenoms = append(collateralDenoms, collateralDenom)
	}

	if !found {
		return nil, types.ErrInvalidCollateralDenom
	}

	params.CollateralDenoms = collateralDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) RemoveCollateralDenom(ctx context.Context, denom string) error {
	params := k.GetParams(ctx)

	collateralDenoms := []types.CollateralDenom{}
	for _, collateralDenom := range params.CollateralDenoms {
		if collateralDenom.DexDenom != denom {
			collateralDenoms = append(collateralDenoms, collateralDenom)
		}
	}

	params.CollateralDenoms = collateralDenoms
	return k.SetParams(ctx, params)
}
