package keeper

import (
	"context"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/swap/types"
)

func (k msgServer) UpdateBurnThreshold(ctx context.Context, req *types.MsgUpdateBurnThreshold) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	burnThreshold, err := math.LegacyNewDecFromStr(req.BurnThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.BurnThreshold = burnThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateMintThreshold(ctx context.Context, req *types.MsgUpdateMintThreshold) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	mintThreshold, err := math.LegacyNewDecFromStr(req.MintThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MintThreshold = mintThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateStakingShare(ctx context.Context, req *types.MsgUpdateStakingShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	stakingShare, err := math.LegacyNewDecFromStr(req.StakingShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.StakingShare = stakingShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateParityFactor(ctx context.Context, req *types.MsgUpdateParityFactor) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	parityFactor, err := math.LegacyNewDecFromStr(req.ParityFactor)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.ParityFactor = parityFactor

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}
