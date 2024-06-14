package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddCollateralDenom(goCtx context.Context, req *types.MsgAddCollateralDenom) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		ltv, err := math.LegacyNewDecFromStr(req.Ltv)
		if err != nil {
			return err
		}

		maxDeposit, ok := math.NewIntFromString(req.MaxDeposit)
		if !ok {
			return fmt.Errorf("invalid max deposit value: %v", req.MaxDeposit)
		}

		collateralDenom := types.CollateralDenom{
			Denom:      req.Denom,
			Ltv:        ltv,
			MaxDeposit: maxDeposit,
		}

		params.CollateralDenoms = append(params.CollateralDenoms, &collateralDenom)

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}
		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateCollateralDenomLTV(goCtx context.Context, req *types.MsgUpdateCollateralDenomLTV) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		ltv, err := math.LegacyNewDecFromStr(req.Ltv)
		if err != nil {
			return err
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
			return types.ErrInvalidCollateralDenom
		}

		params.CollateralDenoms = collateralDenoms

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateCollateralDenomMaxDeposit(goCtx context.Context, req *types.MsgUpdateCollateralDenomMaxDeposit) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		maxDeposit, ok := math.NewIntFromString(req.MaxDeposit)
		if !ok {
			return fmt.Errorf("invalid max deposit value: %v", req.MaxDeposit)
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
			return types.ErrInvalidCollateralDenom
		}

		params.CollateralDenoms = collateralDenoms

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}
