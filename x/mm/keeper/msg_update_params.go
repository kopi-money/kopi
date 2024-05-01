package keeper

import (
	"context"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) UpdateProtocolShare(ctx context.Context, req *types.MsgUpdateProtocolShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	protocolShare, err := math.LegacyNewDecFromStr(req.ProtocolShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.ProtocolShare = protocolShare

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateRedemptionFee(ctx context.Context, req *types.MsgUpdateRedemptionFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	minRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MinRedemptionFee = minRedemptionFee

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateCollateralDiscount(ctx context.Context, req *types.MsgUpdateCollateralDiscount) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	collateralDiscount, err := math.LegacyNewDecFromStr(req.CollateralDiscount)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.CollateralDiscount = collateralDiscount

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateInterestRateParameters(ctx context.Context, req *types.MsgUpdateInterestRateParameters) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	minInterestRate, err := math.LegacyNewDecFromStr(req.MinInterestRate)
	if err != nil {
		return nil, err
	}

	a, err := math.LegacyNewDecFromStr(req.A)
	if err != nil {
		return nil, err
	}

	b, err := math.LegacyNewDecFromStr(req.B)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MinInterestRate = minInterestRate
	params.A = a
	params.B = b

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
