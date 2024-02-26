package keeper

import (
	"context"
	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateTradeFee(ctx context.Context, req *types.MsgUpdateTradeFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	tradeFee, err := math.LegacyNewDecFromStr(req.TradeFee)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.TradeFee = tradeFee

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveShare(ctx context.Context, req *types.MsgUpdateReserveShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	reserveShare, err := math.LegacyNewDecFromStr(req.ReserveShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.ReserveShare = reserveShare

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateVirtualLiquidityDecay(ctx context.Context, req *types.MsgUpdateVirtualLiquidityDecay) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	virtualLiquidityDecay, err := math.LegacyNewDecFromStr(req.VirtualLiquidityDecay)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.VirtualLiquidityDecay = virtualLiquidityDecay

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateFeeReimbursement(ctx context.Context, req *types.MsgUpdateFeeReimbursement) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	feeReimbursement, err := math.LegacyNewDecFromStr(req.FeeReimbursement)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.FeeReimbursement = feeReimbursement

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMaxOrderLife(ctx context.Context, req *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	params.MaxOrderLife = req.MaxOrderLife

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
