package keeper

import (
	"context"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) UpdateProtocolShare(ctx context.Context, req *types.MsgUpdateProtocolShare) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		protocolShare, err := math.LegacyNewDecFromStr(req.ProtocolShare)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.ProtocolShare = protocolShare

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateRedemptionFees(ctx context.Context, req *types.MsgUpdateRedemptionFees) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		minRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
		if err != nil {
			return err
		}

		maxRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.MinRedemptionFee = minRedemptionFee
		params.MaxRedemptionFee = maxRedemptionFee

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateCollateralDiscount(ctx context.Context, req *types.MsgUpdateCollateralDiscount) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		collateralDiscount, err := math.LegacyNewDecFromStr(req.CollateralDiscount)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.CollateralDiscount = collateralDiscount

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateInterestRateParameters(ctx context.Context, req *types.MsgUpdateInterestRateParameters) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		minInterestRate, err := math.LegacyNewDecFromStr(req.MinInterestRate)
		if err != nil {
			return err
		}

		a, err := math.LegacyNewDecFromStr(req.A)
		if err != nil {
			return err
		}

		b, err := math.LegacyNewDecFromStr(req.B)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.MinInterestRate = minInterestRate
		params.A = a
		params.B = b

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}
