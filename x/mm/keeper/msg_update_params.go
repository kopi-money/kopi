package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) UpdateProtocolShare(goCtx context.Context, req *types.MsgUpdateProtocolShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

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

func (k msgServer) UpdateRedemptionFee(goCtx context.Context, req *types.MsgUpdateRedemptionFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

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

func (k msgServer) UpdateCollateralDiscount(goCtx context.Context, req *types.MsgUpdateCollateralDiscount) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

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

func (k msgServer) UpdateInterestRateParameters(goCtx context.Context, req *types.MsgUpdateInterestRateParameters) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

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
