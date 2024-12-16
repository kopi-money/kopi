package keeper

import (
	"context"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/swap/types"
)

func (k msgServer) UpdateBurnThreshold(ctx context.Context, req *types.MsgUpdateBurnThreshold) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		burnThreshold, err := math.LegacyNewDecFromStr(req.BurnThreshold)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.BurnThreshold = burnThreshold

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}
		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateMintThreshold(ctx context.Context, req *types.MsgUpdateMintThreshold) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		mintThreshold, err := math.LegacyNewDecFromStr(req.MintThreshold)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.MintThreshold = mintThreshold

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}
		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateStakingShare(ctx context.Context, req *types.MsgUpdateStakingShare) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		stakingShare, err := math.LegacyNewDecFromStr(req.StakingShare)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.StakingShare = stakingShare

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}
		return nil
	})

	return &types.Void{}, err
}
